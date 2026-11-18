package source

import (
	"context"
	"testing"
	"time"

	"github.com/tidepark/tidetop/internal/feed"
)

func TestDemo(t *testing.T) {
	if DemoLevel(0) != 1 || DemoLevel(DemoPeriod/2) != 0 || DemoLevel(1.3) != DemoLevel(1.3+DemoPeriod) {
		t.Error("DemoLevel")
	}
	bottom := NewDemo("free", time.Now().Add(-DemoPeriod/2*time.Second)).Next(context.Background())
	if !bottom.OK || *feed.Compute(bottom.Feed)[0].Under <= 0 {
		t.Error("바닥에서는 범위 아래다")
	}
	d := NewDemo("free", time.Now())
	d.SetSplit(true)
	if r := d.Next(context.Background()); len(r.Feed.Values) != 8 {
		t.Error("코어 나눔")
	}
	if r := NewDemo("cores", time.Now()).Next(context.Background()); len(r.Feed.Values) != 8 {
		t.Error("cores")
	}
}

func TestLocal(t *testing.T) {
	for _, kind := range []string{"cpu", "free", "mem-cpu", "cores"} {
		r := NewLocal(kind).Next(context.Background())
		if !r.OK {
			t.Fatalf("%s: %s", kind, r.Error)
		}
	}
	l := NewLocal("free")
	l.SetBasis("mem")
	r := l.Next(context.Background())
	if v := r.Feed.Values[0]; !r.OK || v.From == nil || *v.From != Caution {
		t.Error("메모리 기준의 여유")
	}
}

func TestEase(t *testing.T) {
	a := feed.Compute(Demo2(0.2))
	b := feed.Compute(Demo2(0.8))
	if got := Ease(nil, b, 0.1, 0.9); *got[0].Fill != *b[0].Fill {
		t.Error("처음에는 바로 받은 값이다")
	}
	got := Ease(a, b, 0.1, 0.9)
	if f := *got[0].Fill; !(f > *a[0].Fill && f < *b[0].Fill) {
		t.Errorf("조금씩 옮긴다: %v", f)
	}
}

func Demo2(v float64) *feed.Feed {
	return feed.Validate(map[string]any{"spec": 1.0, "values": []any{map[string]any{"value": v, "to": 1.0}}}).Feed
}
