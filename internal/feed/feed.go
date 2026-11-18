// Package feed 는 tidepic 피드의 검사(규칙 3절, 11절)와 계산(규칙 3.2절)이다.
// tidepic 런타임(github.com/tidepark/tidepic 의 runtime/)의 validate.js 와 compute.js 를 옮겼다. tidetop 이 쓰는 항목만 옮기고,
// 그림 컴패니언에만 쓰이는 항목(view, extra, event, step)은 읽지 않는다.
// 맞는 것만 옮겨 담는다. 필수 항목이 틀리면 피드 전체를 버리고, 선택 항목이 틀리면 그 항목만 버린다.
package feed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"
)

// Limits 는 규칙의 한도다.
var Limits = struct{ FeedBytes, Values, Text, Unit, History int }{16 * 1024, 64, 64, 8, 60}

var states = []string{"live", "idle", "done", "fault"}
var formats = []string{"number", "duration"}
var control = regexp.MustCompile("[\u0000-\u001f\u007f]")
var unsafeText = regexp.MustCompile(`[<>]|://`)

// Value 는 검사를 지난 값 하나다. 없는 항목은 nil 이다.
type Value struct {
	Value                   float64
	From, To, Base, Rate    *float64
	Scale, RateScale, Every *float64
	History                 []float64
	State, Unit, Format     string
}

// Meta 는 피드 자체에 대한 정보다.
type Meta struct {
	TTL, At *float64
	Credit  string
}

// Feed 는 검사를 지난 피드다.
type Feed struct {
	Values []Value
	State  string
	Focus  *int
	Meta   *Meta
}

// Result 는 검사의 결과다. OK 가 아니면 Error 와 Code 가 있다.
type Result struct {
	OK    bool
	Feed  *Feed
	Error string
	Code  string
}

func fail(err, code string) Result { return Result{Error: err, Code: code} }

// Parse 는 피드 글을 읽고 검사한다.
func Parse(text []byte) Result {
	if len(text) > Limits.FeedBytes {
		return fail("피드가 16KB 를 넘는다", "too-big")
	}
	dec := json.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	var data any
	if err := dec.Decode(&data); err != nil {
		return fail("피드가 JSON 이 아니다", "not-json")
	}
	// JSON 뒤에 빈칸 말고 다른 것이 있으면 JSON 이 아니다
	if len(bytes.TrimSpace(text[dec.InputOffset():])) > 0 {
		return fail("피드가 JSON 이 아니다", "not-json")
	}
	return Validate(data)
}

// Validate 는 JSON 으로 읽은 피드를 검사한다. 숫자는 json.Number 나 float64 다.
func Validate(data any) Result {
	obj, ok := data.(map[string]any)
	if !ok {
		return fail("피드가 객체가 아니다", "not-object")
	}
	if spec, ok := number(obj["spec"]); !ok || spec != 1 {
		return fail("spec 이 1 이 아니다", "spec")
	}
	list, ok := obj["values"].([]any)
	if !ok || len(list) < 1 {
		return fail("values 가 없다", "no-values")
	}
	if len(list) > Limits.Values {
		return fail("values 가 64개를 넘는다", "too-many-values")
	}
	f := &Feed{State: "live"}
	for i, raw := range list {
		v, ok := cleanValue(raw)
		if !ok {
			return fail(fmt.Sprintf("values[%d].value 가 숫자가 아니다", i), "value")
		}
		f.Values = append(f.Values, v)
	}
	if s, ok := obj["state"].(string); ok && slices.Contains(states, s) {
		f.State = s
	}
	if x, ok := number(obj["focus"]); ok && x == math.Trunc(x) && x >= 0 && int(x) < len(f.Values) {
		n := int(x)
		f.Focus = &n
	}
	if m, ok := obj["meta"].(map[string]any); ok {
		meta := &Meta{}
		for key, dst := range map[string]**float64{"ttl": &meta.TTL, "at": &meta.At} {
			if x, ok := number(m[key]); ok && x >= 0 {
				*dst = &x
			}
		}
		if c, ok := m["credit"].(string); ok && isText(c, Limits.Text) {
			meta.Credit = c
		}
		if meta.TTL != nil || meta.At != nil || meta.Credit != "" {
			f.Meta = meta
		}
	}
	return Result{OK: true, Feed: f}
}

// number 는 유한한 숫자만 받는다(JavaScript 의 Number.isFinite).
func number(x any) (float64, bool) {
	var v float64
	switch n := x.(type) {
	case json.Number:
		f, err := strconv.ParseFloat(string(n), 64)
		if err != nil && !math.IsInf(f, 0) {
			return 0, false
		}
		v = f
	case float64:
		v = n
	case int:
		v = float64(n)
	default:
		return 0, false
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, false
	}
	return v, true
}

// isText: 짧은 글. 제어 문자, HTML 꺾쇠, 주소가 있거나 길면 받지 않는다.
func isText(x string, max int) bool {
	return x != "" && utf8.RuneCountInString(x) <= max && !control.MatchString(x) && !unsafeText.MatchString(x)
}

func cleanValue(raw any) (Value, bool) {
	o, ok := raw.(map[string]any)
	if !ok {
		return Value{}, false
	}
	value, ok := number(o["value"])
	if !ok {
		return Value{}, false
	}
	v := Value{Value: value}
	// 원점과 숫자: 어떤 수든 된다.
	for key, dst := range map[string]**float64{"from": &v.From, "to": &v.To, "base": &v.Base, "rate": &v.Rate} {
		if x, ok := number(o[key]); ok {
			*dst = &x
		}
	}
	// 폭과 간격: 0 보다 커야 한다.
	for key, dst := range map[string]**float64{"scale": &v.Scale, "rateScale": &v.RateScale, "every": &v.Every} {
		if x, ok := number(o[key]); ok && x > 0 {
			*dst = &x
		}
	}
	// 지난 값들. 하나라도 틀리면 줄 전체를 버린다.
	if h, ok := o["history"].([]any); ok && len(h) >= 1 && len(h) <= Limits.History {
		out := make([]float64, 0, len(h))
		for _, x := range h {
			n, ok := number(x)
			if !ok {
				out = nil
				break
			}
			out = append(out, n)
		}
		v.History = out
	}
	if s, ok := o["state"].(string); ok && slices.Contains(states, s) {
		v.State = s
	}
	if u, ok := o["unit"].(string); ok && isText(u, Limits.Unit) {
		v.Unit = u
	}
	if fm, ok := o["format"].(string); ok && slices.Contains(formats, fm) {
		v.Format = fm
	}
	return v, true
}

// Computed 는 계산된 값(규칙 3.2절)이다. 장면은 원래 숫자가 아니라 이것으로 그린다. 낼 수 없는 값은 nil 이다.
type Computed struct {
	Fill, Over, Under, Rise, Momentum, Share *float64
	History                                  []float64
	State                                    string
	Rank                                     *int
	Raw                                      Value
}

func clamp(x, lo, hi float64) float64 { return math.Min(hi, math.Max(lo, x)) }
func ptr(x float64) *float64          { return &x }

// computeValue: 모두 (숫자 − 원점) ÷ 폭이다. total 은 값들의 합이고, 몫을 낼 수 없으면 nil 이다.
func computeValue(v Value, total *float64) Computed {
	from := 0.0
	if v.From != nil {
		from = *v.From
	}
	span := 0.0
	if v.To != nil {
		span = *v.To - from
	}
	c := Computed{Raw: v}
	if span != 0 {
		along := (v.Value - from) / span
		c.Fill = ptr(clamp(along, 0, 1))
		c.Over = ptr(clamp(along-1, 0, 1))
		c.Under = ptr(clamp(-along, 0, 1))
	}
	scale := math.Abs(span)
	if v.Scale != nil {
		scale = *v.Scale
	}
	if v.Base != nil && scale > 0 {
		c.Rise = ptr(clamp((v.Value-*v.Base)/scale, -1, 1))
	}
	if v.Rate != nil && v.RateScale != nil && *v.RateScale > 0 {
		c.Momentum = ptr(clamp(*v.Rate / *v.RateScale, -1, 1))
	}
	if v.History != nil {
		points := append(append([]float64{}, v.History...), v.Value)
		if span != 0 {
			for i, x := range points {
				points[i] = clamp((x-from)/span, 0, 1)
			}
		} else {
			lo, hi := slices.Min(points), slices.Max(points)
			for i, x := range points {
				if hi > lo {
					points[i] = (x - lo) / (hi - lo)
				} else {
					points[i] = 0.5
				}
			}
		}
		c.History = points
	}
	if total != nil {
		c.Share = ptr(v.Value / *total)
	}
	return c
}

// Compute 는 피드의 값마다 계산된 값을 낸다.
func Compute(f *Feed) []Computed {
	many := len(f.Values) > 1
	sum := 0.0
	nonNegative := true
	for _, v := range f.Values {
		sum += v.Value
		if v.Value < 0 {
			nonNegative = false
		}
	}
	var total *float64
	if many && sum > 0 && nonNegative {
		total = &sum
	}
	out := make([]Computed, len(f.Values))
	for i, v := range f.Values {
		c := computeValue(v, total)
		c.State = v.State
		if c.State == "" {
			c.State = f.State
		}
		if many {
			rank := 1
			for _, o := range f.Values {
				if o.Value > v.Value {
					rank++
				}
			}
			c.Rank = &rank
		}
		out[i] = c
	}
	return out
}
