// Package art 는 사용자가 만든 글자 그림 폴더를 장면으로 쓴다.
//
// 폴더 안의 <숫자>.txt 가 그림이다. 「값이 이 숫자 이상이면 이 그림」이다.
//   - 숫자는 값의 단위 그대로다. 0~10 이면 0.txt~10.txt, 켜짐·꺼짐이면 0.txt 와 1.txt, 퍼센트면 0~100.
//   - 값이 가장 작은 숫자보다 작으면 가장 작은 그림을 쓴다. 숫자가 아닌 이름의 파일은 읽지 않는다.
//   - 한 파일에 `---` 만 있는 줄로 여러 장을 나누면 1초에 두 장씩 넘긴다. 빠르기는 값과 상관없다.
//   - 그림은 화면 가운데에 놓고, 넘치면 자른다.
//
// 파일 속의 터미널 제어 글자(색, 커서, 클립보드 쓰기 따위)는 모두 걸러 낸다. 화면에는 글자만 간다.
// 폴더는 켤 때 한 번만 읽는다.
package art

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/tidepark/tidetop/internal/glitch"
	"github.com/tidepark/tidetop/internal/noise"
	"github.com/tidepark/tidetop/internal/render"
)

var limits = struct{ files, bytes, frames, lines, width int }{200, 64 * 1024, 32, 200, 400}

const pace = 2 // 1초에 넘기는 장 수

var messages = map[string]map[string]string{
	"ko": {
		"folder": "그림 폴더를 읽을 수 없다: %s",
		"empty":  "그림 폴더에 <숫자>.txt 파일이 없다: %s",
		"big":    "그림 파일이 64KB 를 넘는다: %s",
	},
	"en": {
		"folder": "cannot read the art folder: %s",
		"empty":  "the art folder has no <number>.txt files: %s",
		"big":    "an art file is larger than 64KB: %s",
	},
}

var (
	osc     = regexp.MustCompile(`\x1b\][\s\S]*?(?:\x07|\x1b\\|$)`)
	dcs     = regexp.MustCompile(`\x1b[P^_X][\s\S]*?(?:\x1b\\|$)`)
	csi     = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	escTwo  = regexp.MustCompile(`\x1b[\s\S]?`)
	ctrl    = regexp.MustCompile("[\x00-\x08\x0b-\x1f\x7f-\u009f]")
	numName = regexp.MustCompile(`(?i)^(-?\d+(?:\.\d+)?)\.txt$`)
)

// Clean 은 터미널 제어 글자를 걸러 내고 칸에 넣을 수 있는 글자만 남긴다.
func Clean(text string) string {
	// 이스케이프 열: OSC(BEL 이나 ST 로 끝남), DCS·APC·PM·SOS, CSI, 그 밖의 두 글자 열
	text = osc.ReplaceAllString(text, "")
	text = dcs.ReplaceAllString(text, "")
	text = csi.ReplaceAllString(text, "")
	text = escTwo.ReplaceAllString(text, "")
	// 8비트 제어 글자(C1)와 남은 제어 글자. 줄바꿈과 탭만 남긴다
	text = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(text)
	text = ctrl.ReplaceAllString(text, "")
	var b strings.Builder
	for _, r := range text {
		switch {
		// 폭이 없는 글자와 결합 글자는 칸을 어긋나게 하므로 뺀다
		case unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf):
		// 한 칸에 담을 수 없는 글자(이모지 등)는 ? 로 바꾼다
		case r > 0xffff:
			b.WriteRune('?')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// 줄 하나를 칸의 글자 번호로 바꾼다. 두 칸 글자 뒤에는 Covered 를 둔다.
func cellsOf(line string) []uint16 {
	var cells []uint16
	for _, r := range strings.ReplaceAll(line, "\t", "    ") {
		if len(cells) >= limits.width {
			break
		}
		cells = append(cells, uint16(r))
		if render.IsWide(r) {
			cells = append(cells, render.Covered)
		}
	}
	return cells
}

func framesOf(text string) [][][]uint16 {
	var frames [][][]uint16
	var lines [][]uint16
	for _, line := range strings.Split(Clean(text), "\n") {
		if strings.TrimSpace(line) == "---" {
			frames = append(frames, lines)
			lines = nil
		} else if len(lines) < limits.lines {
			lines = append(lines, cellsOf(line))
		}
	}
	frames = append(frames, lines)
	var out [][][]uint16
	for _, f := range frames {
		// 끝의 빈 줄은 버린다
		for len(f) > 0 && len(f[len(f)-1]) == 0 {
			f = f[:len(f)-1]
		}
		if len(f) > 0 && len(out) < limits.frames {
			out = append(out, f)
		}
	}
	return out
}

// Picture 는 그림 하나다. At 이상의 값에서 보인다.
type Picture struct {
	At     float64
	Frames [][][]uint16
}

// Load 는 폴더를 읽어 장면을 만든다. lang 은 "ko" 나 "en" 이다(잘못을 알리는 말).
func Load(dir, lang string) (*Scene, error) {
	say := messages[lang]
	if say == nil {
		say = messages["ko"]
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf(say["folder"], dir)
	}
	var pictures []Picture
	for _, e := range entries {
		m := numName.FindStringSubmatch(e.Name())
		if m == nil || len(pictures) >= limits.files {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if info.Size() > int64(limits.bytes) {
			return nil, fmt.Errorf(say["big"], e.Name())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		at, _ := strconv.ParseFloat(m[1], 64)
		if frames := framesOf(string(data)); len(frames) > 0 {
			pictures = append(pictures, Picture{at, frames})
		}
	}
	if len(pictures) == 0 {
		return nil, fmt.Errorf(say["empty"], dir)
	}
	sort.SliceStable(pictures, func(i, j int) bool { return pictures[i].At < pictures[j].At })
	return &Scene{Pictures: pictures}, nil
}

// Pick 은 값에 맞는 그림이다: 값 이하의 숫자 중 가장 큰 것. 값이 모두보다 작으면 가장 작은 것.
func Pick(pictures []Picture, value float64) Picture {
	chosen := pictures[0]
	for _, p := range pictures {
		if p.At <= value {
			chosen = p
		}
	}
	return chosen
}

// Scene 은 글자 그림 장면이다. 이 컴퓨터의 값을 쓸 때는 CPU 사용률(%)을 읽는다.
type Scene struct{ Pictures []Picture }

func (*Scene) Name() string    { return "art" }
func (*Scene) Local() string   { return "cpu" }
func (*Scene) Skins() []string { return nil }

type ctx struct {
	lines  [][]uint16
	mood   string
	stale  bool
	glitch []glitch.Line
}

func (s *Scene) Frame(in render.Input) render.Ctx {
	var v struct {
		state string
		value float64
		under float64
	}
	if len(in.Values) > 0 {
		v.state, v.value = in.Values[0].State, in.Values[0].Raw.Value
		if in.Values[0].Under != nil {
			v.under = *in.Values[0].Under
		}
	}
	mood := "rest"
	if v.state == "fault" || v.state == "live" {
		mood = v.state
	}
	p := Pick(s.Pictures, v.value)
	t := 0.0
	if mood == "live" {
		t = in.Time
	}
	c := &ctx{lines: p.Frames[int(noise.Mod(math.Floor(t*pace), float64(len(p.Frames))))], mood: mood, stale: in.State == "stale"}
	if mood == "live" {
		// 범위 아래(under)로 내려가면 다른 장면처럼 깨진 모니터 같은 줄이 간다
		c.glitch = glitch.Make(noise.Smooth(0, 0.1, v.under), in.W, in.H, t, 0)
	}
	return c
}

func (c *ctx) Pixel(int, int) (noise.RGB, bool)    { return noise.RGB{}, false }
func (c *ctx) Glyph(int, int) (rune, string, bool) { return 0, "", false }
func (c *ctx) Dots(render.Dot)                     {}
func (c *ctx) Box() *render.Box                    { return nil }
func (c *ctx) Glitch() []glitch.Line               { return c.glitch }

// Cells 는 판의 칸에 글자를 바로 넣는다.
func (c *ctx) Cells(g *render.Grid, color bool) {
	width := 0
	for _, l := range c.lines {
		width = max(width, len(l))
	}
	x0 := int(math.Floor(float64(g.Cols-width) / 2))
	y0 := int(math.Floor(float64(g.Rows-len(c.lines)) / 2))
	// 쉼은 흐리게, 고장과 끊김은 잿빛으로
	fg := render.None
	if color {
		if c.mood == "fault" || c.stale {
			fg = render.Basic + 37
		} else if c.mood == "rest" {
			fg = render.Basic + 90
		}
	}
	for row, line := range c.lines {
		y := y0 + row
		if y < 0 || y >= g.Rows {
			continue
		}
		for k, code := range line {
			x := x0 + k
			if x < 0 || x >= g.Cols {
				continue
			}
			// 오른쪽 끝에서 두 칸 글자가 잘리면 빈칸으로 둔다
			cut := x == g.Cols-1 && k+1 < len(line) && line[k+1] == render.Covered
			// 왼쪽 끝에서 두 칸 글자의 앞이 잘리면 남은 칸도 빈칸으로 둔다
			orphan := code == render.Covered && x == 0
			i := y*g.Cols + x
			g.Ch[i] = code
			if cut || orphan {
				g.Ch[i] = 32
			}
			if code != 32 {
				g.Fg[i] = fg
			}
		}
	}
}
