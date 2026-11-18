package scenes

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/tidepark/tidetop/internal/feed"
	"github.com/tidepark/tidetop/internal/render"
)

// 기준 묶음(testdata/frames.json.gz)의 한 장
type frameCase struct {
	Scene     string
	Skin      string
	Cols      int
	Rows      int
	Values    []any
	Quality   string
	Time      float64
	FeedState string
	State     string
	Color     *bool
	Tiles     bool
	Crop      map[string]int
	Note      []string
	At        string
	Lines     []string
}

type cell struct {
	ch     rune
	fg, bg string
}

var sgr = regexp.MustCompile(`^\x1b\[([0-9;]*)m`)

// 화면 글을 칸으로 푼다. 색 부호는 그 뒤 칸들의 글자색·바탕색이 된다.
func cells(lines []string) [][]cell {
	out := make([][]cell, len(lines))
	for y, line := range lines {
		fg, bg := "", ""
		rs := []rune(line)
		for i := 0; i < len(rs); {
			if rs[i] == 0x1b {
				m := sgr.FindStringSubmatch(string(rs[i:]))
				code := m[1]
				switch {
				case code == "0":
					fg, bg = "", ""
				case code == "49":
					bg = ""
				case len(code) > 3 && code[:3] == "48;":
					bg = code
				default:
					fg = code
				}
				i += len([]rune(m[0]))
				continue
			}
			out[y] = append(out[y], cell{rs[i], fg, bg})
			i++
		}
	}
	return out
}

var registry = map[string]render.Scene{"campfire": Campfire{}, "tide": Tide{}, "windows": Windows{}, "grove": Grove{}, "stars": Stars{}, "library": Library{}}

func loadFrames(t *testing.T) []frameCase {
	f, err := os.Open("../../testdata/frames.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var cases []frameCase
	if err := json.NewDecoder(z).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

// 칸마다 견준다. 장마다 99.9% 넘게 같아야 한다.
func TestFramesLikeJavaScript(t *testing.T) {
	worst, total, frames := 1.0, 0, 0
	for _, c := range loadFrames(t) {
		s, ok := registry[c.Scene]
		if !ok {
			continue
		}
		if c.Crop != nil {
			s = render.Crop{Scene: s, Across: c.Crop["across"], Down: c.Crop["down"], Col: c.Crop["col"], Row: c.Crop["row"]}
		}
		state := c.FeedState
		if state == "" {
			state = "live"
		}
		r := feed.Validate(map[string]any{"spec": 1.0, "values": c.Values, "state": state})
		if !r.OK {
			t.Fatalf("피드: %s", r.Error)
		}
		in := render.Input{
			Values: feed.Compute(r.Feed), State: "live", Cols: c.Cols, Rows: c.Rows, Time: c.Time,
			Quality: c.Quality, Color: c.Color == nil || *c.Color, Skin: c.Skin,
		}
		if c.State != "" {
			in.State = c.State
		}
		var g *render.Grid
		if c.Tiles {
			g = render.RenderTiles(s, in)
		} else {
			g = render.Render(s, in)
		}
		if c.Note != nil {
			render.Stamp(g, c.Note, c.Quality == "high", true, c.At)
		}
		got := cells(render.Lines(g))
		want := cells(c.Lines)
		same, n := 0, 0
		var first [2]int
		firstSet := false
		for y := range want {
			for x := range want[y] {
				n++
				if y < len(got) && x < len(got[y]) && got[y][x] == want[y][x] {
					same++
				} else if !firstSet {
					first, firstSet = [2]int{x, y}, true
				}
			}
			if len(got[y]) != len(want[y]) {
				t.Errorf("%s %dx%d %s t=%v: %d번 줄의 칸 수 %d, JavaScript %d", c.Scene, c.Cols, c.Rows, c.Quality, c.Time, y, len(got[y]), len(want[y]))
			}
		}
		k := float64(same) / float64(n)
		frames++
		total += n - same
		if k < worst {
			worst = k
		}
		if k < 0.999 {
			t.Errorf("%s %s %dx%d %s t=%v 값=%v 상태=%s/%s: %.4f 같음, 처음 다른 칸 %v", c.Scene, c.Skin, c.Cols, c.Rows, c.Quality, c.Time, c.Values, state, c.State, k, first)
		}
	}
	t.Logf("장 %d, 다른 칸 모두 %d, 가장 낮은 같음 %.5f", frames, total, worst)
}
