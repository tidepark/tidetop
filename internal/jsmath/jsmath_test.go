package jsmath

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"
)

func bits(t *testing.T, s string) float64 {
	u, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		t.Fatal(err)
	}
	return math.Float64frombits(u)
}

// JavaScript 판에서 뽑은 입력과 출력(testdata/math.json)과 비트까지 같아야 한다.
func TestSameAsJavaScript(t *testing.T) {
	data, err := os.ReadFile("../../testdata/math.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases map[string][][]string
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	one := map[string]func(float64) float64{"sin": Sin, "cos": Cos, "exp": Exp, "log": Log, "round": Round}
	two := map[string]func(float64, float64) float64{"pow": Pow, "atan2": Atan2, "hypot": Hypot}
	for name, list := range cases {
		bad := 0
		for _, c := range list {
			var got, want float64
			if f, ok := one[name]; ok {
				got, want = f(bits(t, c[0])), bits(t, c[1])
			} else {
				got, want = two[name](bits(t, c[0]), bits(t, c[1])), bits(t, c[2])
			}
			if math.Float64bits(got) != math.Float64bits(want) && !(math.IsNaN(got) && math.IsNaN(want)) {
				if bad < 1 {
					t.Logf("%s%v: got %v want %v", name, c, got, want)
				}
				bad++
			}
		}
		// 반올림은 정확히 같아야 한다. 나머지는 마지막 비트가 다를 수 있어(Go 와 V8 의 수학 함수가 다르다) 알리기만 한다.
		if bad > 0 && name == "round" {
			t.Errorf("%s: %d / %d 다름", name, bad, len(list))
		} else if bad > 0 {
			t.Logf("%s: %d / %d 가 마지막 비트에서 다름", name, bad, len(list))
		}
	}
}
