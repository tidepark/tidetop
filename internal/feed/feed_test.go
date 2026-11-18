package feed

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// JavaScript 의 computeFeed 결과와 같은 모양(JSON)으로 바꾼다
func asJS(c Computed) map[string]any {
	f := func(p *float64) any {
		if p == nil {
			return nil
		}
		return *p
	}
	raw := map[string]any{"value": c.Raw.Value}
	for key, p := range map[string]*float64{"from": c.Raw.From, "to": c.Raw.To, "base": c.Raw.Base, "rate": c.Raw.Rate, "scale": c.Raw.Scale, "rateScale": c.Raw.RateScale, "every": c.Raw.Every} {
		if p != nil {
			raw[key] = *p
		}
	}
	if c.Raw.History != nil {
		h := []any{}
		for _, x := range c.Raw.History {
			h = append(h, x)
		}
		raw["history"] = h
	}
	for key, s := range map[string]string{"state": c.Raw.State, "unit": c.Raw.Unit, "format": c.Raw.Format} {
		if s != "" {
			raw[key] = s
		}
	}
	var history any
	if c.History != nil {
		h := []any{}
		for _, x := range c.History {
			h = append(h, x)
		}
		history = h
	}
	var rank any
	if c.Rank != nil {
		rank = float64(*c.Rank)
	}
	return map[string]any{
		"fill": f(c.Fill), "over": f(c.Over), "under": f(c.Under), "rise": f(c.Rise), "momentum": f(c.Momentum),
		"history": history, "share": f(c.Share), "raw": raw, "state": c.State, "rank": rank,
	}
}

func TestSameAsJavaScript(t *testing.T) {
	data, err := os.ReadFile("../../testdata/feeds.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Text   string
		OK     bool
		Error  *string
		Values []map[string]any
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		r := Parse([]byte(c.Text))
		if r.OK != c.OK {
			t.Errorf("%s: ok %v, JavaScript %v (%s)", c.Text, r.OK, c.OK, r.Error)
			continue
		}
		if !r.OK {
			if r.Error != *c.Error {
				t.Errorf("%s: %q, JavaScript %q", c.Text, r.Error, *c.Error)
			}
			continue
		}
		got := Compute(r.Feed)
		for i, want := range c.Values {
			if g := asJS(got[i]); !reflect.DeepEqual(g, want) {
				t.Errorf("%s [%d]:\n got %v\nwant %v", c.Text, i, g, want)
			}
		}
	}
}
