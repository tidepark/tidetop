// Package source 는 장면에 줄 값을 얻는다: 이 컴퓨터의 CPU·메모리, 흉내 값, tidepic 피드(파일, https).
// 어느 쪽이든 피드 검사(feed.Validate)를 거친다.
package source

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/tidepark/tidetop/internal/feed"
)

// Caution 은 여유를 읽는 장면에 주는 범위의 아래 끝(%)이다. 여유가 이보다 적으면(CPU 90% 넘음) 범위 아래(under)이고 주의를 그린다.
const Caution = 10

// Source 는 값을 얻는 곳이다.
type Source interface {
	Next(ctx context.Context) feed.Result
	// Every 는 기본 주기, Floor 는 이보다 자주 묻지 않는 바닥이다(https 피드만 있다).
	Every() time.Duration
	Floor() time.Duration
}

// Switchable 은 키로 바꿀 수 있는 출처다(이 컴퓨터, 흉내 값).
type Switchable interface {
	Source
	SetBasis(basis string) // "cpu" 또는 "mem"
	Basis() string
	SetSplit(split bool) // 값 하나 대신 코어마다의 값
	Split() bool
}

func percent(x float64) float64 { return math.Round(math.Min(1, math.Max(0, x))*1000) / 10 }

func one(x float64) map[string]any {
	return map[string]any{"value": percent(x), "to": 100.0, "unit": "%"}
}

func free(x float64) map[string]any {
	v := one(1 - x)
	v["from"] = float64(Caution)
	return v
}

// shape 은 사용률(0~1)들을 장면이 바라는 값으로 바꾼다.
// kind: "cpu"(사용률 하나), "free"(여유 하나), "mem-cpu"(메모리, CPU), "cores"(코어마다)
func shape(kind, basis string, split bool, cpuAll float64, cores []float64, memory float64) []any {
	limit := func(l []float64) []float64 {
		if len(l) > feed.Limits.Values {
			return l[:feed.Limits.Values]
		}
		return l
	}
	var out []any
	switch {
	case kind == "cores":
		for _, c := range limit(cores) {
			out = append(out, one(c))
		}
	case kind == "mem-cpu":
		out = []any{one(memory), one(cpuAll)}
	default:
		each := one
		if kind == "free" {
			each = free
		}
		if split {
			for _, c := range limit(cores) {
				out = append(out, each(c))
			}
		} else if basis == "mem" {
			out = []any{each(memory)}
		} else {
			out = []any{each(cpuAll)}
		}
	}
	return out
}

func result(values []any) feed.Result {
	return feed.Validate(map[string]any{"spec": 1.0, "values": values})
}

// Local 은 이 컴퓨터의 CPU 와 메모리다.
type Local struct {
	Kind     string
	basis    string
	split    bool
	before   []cpu.TimesStat
	memory   float64
	memoryAt time.Time
}

func NewLocal(kind string) *Local { return &Local{Kind: kind, basis: "cpu"} }

func (l *Local) Every() time.Duration { return time.Second }
func (l *Local) Floor() time.Duration { return 0 }
func (l *Local) SetBasis(b string)    { l.basis = b }
func (l *Local) Basis() string        { return l.basis }
func (l *Local) SetSplit(s bool)      { l.split = s }
func (l *Local) Split() bool          { return l.split }

// 코어마다 지금까지 쓴 시간을 두 번 재서 그 사이를 본다
func usage(a, b []cpu.TimesStat) (float64, []float64) {
	idle, total := 0.0, 0.0
	cores := make([]float64, len(b))
	for i, now := range b {
		busyAll := func(t cpu.TimesStat) (float64, float64) {
			id := t.Idle + t.Iowait
			return id, id + t.User + t.Nice + t.System + t.Irq + t.Softirq + t.Steal
		}
		ni, nt := busyAll(now)
		var pi, pt float64
		if i < len(a) {
			pi, pt = busyAll(a[i])
		}
		di, dt := ni-pi, nt-pt
		idle += di
		total += dt
		if dt > 0 {
			cores[i] = 1 - di/dt
		}
	}
	if total <= 0 {
		return 0, cores
	}
	return 1 - idle/total, cores
}

func (l *Local) Next(ctx context.Context) feed.Result {
	if l.before == nil {
		l.before, _ = cpu.TimesWithContext(ctx, true)
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
		}
	}
	now, err := cpu.TimesWithContext(ctx, true)
	if err != nil {
		return feed.Result{Error: err.Error(), Code: "cpu"}
	}
	all, cores := usage(l.before, now)
	l.before = now
	useMemory := l.Kind == "mem-cpu" || (l.basis == "mem" && !l.split)
	wait := 1500 * time.Millisecond
	if l.Kind == "mem-cpu" {
		wait = 5 * time.Second
	}
	if useMemory && (l.memoryAt.IsZero() || time.Since(l.memoryAt) > wait) {
		if v, ok := memoryUsed(ctx); ok {
			l.memory = v
		}
		l.memoryAt = time.Now()
	}
	return result(shape(l.Kind, l.basis, l.split, all, cores, l.memory))
}

// DemoPeriod 는 흉내 값이 한 번 오르내리는 주기(초)다.
const DemoPeriod = 40

// DemoLevel: 40초에 한 번 넉넉함 → 바닥 → 넉넉함. 시각만의 함수다.
func DemoLevel(t float64) float64 {
	p := math.Mod(math.Mod(t/DemoPeriod, 1)+1, 1)
	switch {
	case p < 0.05 || p >= 0.95:
		return 1
	case p < 0.45:
		return 1 - (p-0.05)/0.4
	case p < 0.55:
		return 0
	}
	return (p - 0.55) / 0.4
}

// Demo 는 흉내 값이다. 코어는 여덟 개이고 서로 5초씩 어긋난다. 메모리 기준은 반 주기 어긋난 값이다.
type Demo struct {
	Kind    string
	Started time.Time // 이어 붙인 창은 벽시계(유닉스 시각 0)를 기준으로 한다
	basis   string
	split   bool
}

func NewDemo(kind string, started time.Time) *Demo {
	return &Demo{Kind: kind, Started: started, basis: "cpu"}
}

func (d *Demo) Every() time.Duration { return 500 * time.Millisecond }
func (d *Demo) Floor() time.Duration { return 0 }
func (d *Demo) SetBasis(b string)    { d.basis = b }
func (d *Demo) Basis() string        { return d.basis }
func (d *Demo) SetSplit(s bool)      { d.split = s }
func (d *Demo) Split() bool          { return d.split }

func (d *Demo) Next(context.Context) feed.Result {
	t := float64(time.Since(d.Started).Milliseconds()) / 1000
	// 바닥에서는 사용률이 105% 까지 가서, 여유(free)가 범위(10~100) 아래로 내려간다
	used := func(k float64) float64 { return math.Min(1.05, (1-DemoLevel(t+k))*1.05) }
	cores := make([]float64, 8)
	for i := range cores {
		cores[i] = used(float64(i) * 5)
	}
	all := used(0)
	if d.basis == "mem" {
		all = used(DemoPeriod / 2)
	}
	memory := all
	if d.Kind == "mem-cpu" {
		memory = 0.4 + 0.3*math.Sin(t/9)
	}
	return result(shape(d.Kind, d.basis, d.split, all, cores, memory))
}

var httpsAddr = regexp.MustCompile(`(?i)^https://`)
var otherScheme = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)

// Data 는 tidepic 피드다. 파일은 1초마다 다시 읽고, https 주소는 10초보다 자주 묻지 않는다
// (보내는 쪽이 적은 meta.ttl 이 더 길면 그것을 따른다).
type Data struct {
	target string
	remote bool
	every  time.Duration
	client *http.Client
}

// NewData 는 파일 경로나 https 주소를 받는다. 다른 주소는 받지 않는다.
func NewData(target string) (*Data, error) {
	if httpsAddr.MatchString(target) {
		return &Data{target: target, remote: true, every: 10 * time.Second, client: &http.Client{Timeout: 8 * time.Second}}, nil
	}
	if otherScheme.MatchString(target) {
		return nil, fmt.Errorf("--data 는 파일 경로나 https 주소만 받는다")
	}
	return &Data{target: target, every: time.Second}, nil
}

func (d *Data) Every() time.Duration { return d.every }
func (d *Data) Floor() time.Duration {
	if d.remote {
		return d.every
	}
	return 0
}

func (d *Data) Next(ctx context.Context) feed.Result {
	if !d.remote {
		text, err := os.ReadFile(d.target)
		if err != nil {
			return feed.Result{Error: err.Error(), Code: "read"}
		}
		return feed.Parse(text)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.target, nil)
	if err != nil {
		return feed.Result{Error: err.Error(), Code: "fetch"}
	}
	req.Header.Set("Cache-Control", "no-store")
	res, err := d.client.Do(req)
	if err != nil {
		return feed.Result{Error: err.Error(), Code: "fetch"}
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return feed.Result{Error: fmt.Sprintf("피드 응답이 %d 이다", res.StatusCode), Code: "status"}
	}
	if res.ContentLength > int64(feed.Limits.FeedBytes) {
		return feed.Result{Error: "피드가 16KB 를 넘는다", Code: "too-big"}
	}
	text, err := io.ReadAll(io.LimitReader(res.Body, int64(feed.Limits.FeedBytes)+1))
	if err != nil {
		return feed.Result{Error: err.Error(), Code: "fetch"}
	}
	r := feed.Parse(text)
	// 보내는 쪽이 적은 간격(meta.ttl, 초)이 더 길면 그것을 따른다
	if r.OK {
		ttl := 0.0
		if r.Feed.Meta != nil && r.Feed.Meta.TTL != nil {
			ttl = *r.Feed.Meta.TTL
		}
		d.every = time.Duration(math.Max(10, ttl) * float64(time.Second))
	}
	return r
}

// Ease 는 받은 값을 부드럽게 따라간다(컴패니언에서는 런타임이 하는 일이다). 장면은 받은 값을 그대로 그린다.
func Ease(shown, target []feed.Computed, dt, seconds float64) []feed.Computed {
	if shown == nil || len(shown) != len(target) {
		return target
	}
	k := 1 - math.Exp(-dt/seconds)
	out := make([]feed.Computed, len(target))
	for i, to := range target {
		o := to
		from := shown[i]
		step := func(f, t *float64) *float64 {
			if f == nil || t == nil {
				return t
			}
			v := *f + (*t-*f)*k
			return &v
		}
		o.Fill, o.Over, o.Under = step(from.Fill, to.Fill), step(from.Over, to.Over), step(from.Under, to.Under)
		o.Rise, o.Momentum, o.Share = step(from.Rise, to.Rise), step(from.Momentum, to.Momentum), step(from.Share, to.Share)
		out[i] = o
	}
	return out
}
