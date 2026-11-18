package source

import (
	"context"
	"os"
	"path/filepath"
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

// 파일 피드: meta.ttl 이 있고 파일이 ttl 의 세 배보다 오래 그대로면 끊김(old), 아니면 그대로
func TestFileFeedGoesStaleWhenOld(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string, age time.Duration) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-age)
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
		return p
	}
	withTTL := `{"spec":1,"values":[{"value":42,"to":100}],"meta":{"ttl":5}}`
	noTTL := `{"spec":1,"values":[{"value":42,"to":100}]}`
	cases := []struct {
		name, text string
		age        time.Duration
		old        bool
	}{
		{"ttl-old.json", withTTL, 16 * time.Second, true},
		{"ttl-new.json", withTTL, 14 * time.Second, false},
		{"plain-old.json", noTTL, 24 * time.Hour, false},
	}
	for _, c := range cases {
		d, err := NewData(write(c.name, c.text, c.age))
		if err != nil {
			t.Fatal(err)
		}
		r := d.Next(context.Background())
		if !r.OK {
			t.Fatalf("%s: 값을 읽지 못함: %s", c.name, r.Error)
		}
		if got := r.Code == "old"; got != c.old {
			t.Errorf("%s: 끊김 %v, 바란 것 %v", c.name, got, c.old)
		}
		if got := d.Old() > 0; got != c.old {
			t.Errorf("%s: Old() %v", c.name, d.Old())
		}
	}
}
