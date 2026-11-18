// tidetop: 터미널에서 도는 tidepic 장면. 정확한 숫자보다 지금의 분위기를 읽는 화면이다.
package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tidepark/tidetop/internal/art"
	"github.com/tidepark/tidetop/internal/feed"
	"github.com/tidepark/tidetop/internal/render"
	"github.com/tidepark/tidetop/internal/scenes"
	"github.com/tidepark/tidetop/internal/source"
	"github.com/tidepark/tidetop/internal/term"
)

var registry = map[string]render.Scene{
	"campfire": scenes.Campfire{}, "grove": scenes.Grove{}, "library": scenes.Library{}, "stars": scenes.Stars{}, "tide": scenes.Tide{}, "windows": scenes.Windows{},
}
var sceneNames = []string{"grove", "stars", "library", "campfire", "tide", "windows"}

const defaultScene = "grove"

// version 은 빌드할 때 -ldflags "-X main.version=..." 로 넣는다
var version = "dev"

// Tab, n 으로 넘기는 순서와 이름
var sceneOrder = []string{"grove", "stars", "library", "campfire", "tide", "windows"}
var sceneTitles = map[string]string{"grove": "숲", "stars": "별하늘", "library": "도서관", "campfire": "모닥불", "tide": "물결", "windows": "불 켜진 창"}

var korean = regexp.MustCompile(`(?i)^ko`).MatchString(firstEnv("LC_ALL", "LC_MESSAGES", "LANG"))

func firstEnv(names ...string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}

const helpKo = `tidetop — 터미널에서 도는 tidepic 장면

쓰는 법: tidetop [옵션]
        tidetop init <폴더>   글자 그림 폴더의 견본(모닥불 0.txt~100.txt)을 만든다

  --scene <이름>       장면: {{scenes}} (기본 {{default}})
  --data <파일|주소>   tidepic 피드를 읽는다. 파일 경로나 https 주소.
                       없으면 이 컴퓨터의 CPU 와 메모리를 읽는다.
  --art <폴더>         <숫자>.txt 글자 그림 폴더를 장면으로 쓴다. 값이 그 숫자 이상이면 그 그림.
                       --data 가 없으면 숫자는 이 컴퓨터의 CPU 사용률(%)이다
  --demo               실제 수치 대신 40초마다 오르내리는 흉내 값을 쓴다
  --skin <이름>        장면의 겉모습. grove: night(기본), day. stars: milkyway(기본), galaxy
  --quality low|high   high: 24비트 색과 UTF-8, low: ASCII. 없으면 터미널을 보고 고른다.
  --fps <수>           1초에 그릴 장 수 (기본 high 15, SSH 너머 8, low 2)
  -M, --mem            메모리 기준으로 시작한다(값 하나를 읽는 장면). 키 m 과 같다
  -t, --top            여유가 문턱 아래로 이어지면 tidetop 을 닫고 top 을 띄운다. 돌아오지 않는다
  -m, --top-below <%>  -t 의 문턱. 보고 있는 여유(%, CPU 또는 메모리) 가 이 값 이하이면 (기본 20)
  -d, --top-after <초> -t 가 기다리는 시간. 문턱 아래가 이만큼 이어지면 넘어간다 (기본 5, 0 이면 바로)
  --grid <가로>x<세로> 여러 창을 이어 붙여 그림 하나를 만든다. 큰 그림을 이만큼 조각으로 나눈다 (grove, stars, library)
  --pos <번호>         이 창이 그릴 조각. 왼쪽 위부터 1, 2, … (2x2 이면 1 2 / 3 4)
  --once               한 장만 찍고 끝낸다
  --help               이 글
  --version            판 번호를 보인다

키 (대소문자를 가리지 않는다)
  Tab, n 장면 바꾸기    스페이스 지금 읽기    s 읽는 주기    p / m CPU / 메모리    1, c 코어마다 나눠 보기
  z 스킨    i 상태 줄    t 바로 top 으로    h, ? 도움말    q, Ctrl+C 끝내기

자세한 것: man tidetop
`

const helpEn = `tidetop — a tidepic scene in your terminal

Usage: tidetop [options]
       tidetop init <folder>   write a sample text picture folder (a campfire, 0.txt to 100.txt)

  --scene <name>       scene: {{scenes}} (default {{default}})
  --data <file|url>    read a tidepic feed from a file path or an https URL.
                       Without it, reads this computer's CPU and memory.
  --art <folder>       use a folder of <number>.txt text pictures as the scene. A picture shows
                       when the value is at least its number. Without --data the number is CPU use (%)
  --demo               use a pretend value that rises and falls every 40 seconds
  --skin <name>        the scene's look. grove: night (default), day. stars: milkyway (default), galaxy
  --quality low|high   high: 24-bit color and UTF-8, low: ASCII. Picked from the terminal if omitted.
  --fps <n>            frames per second (default 15 for high, 8 over SSH, 2 for low)
  -M, --mem            start on memory instead of CPU (scenes that read one value). Same as key m
  -t, --top            close tidetop and open top when headroom stays below a line. It does not come back
  -m, --top-below <%>  the line for -t: the headroom shown (%, CPU or memory) at or below this (default 20)
  -d, --top-after <s>  how long -t waits below the line before switching (default 5, 0 switches at once)
  --grid <a>x<d>       join several windows into one picture, split into a x d pieces (grove, stars, library)
  --pos <n>            the piece this window draws, from the top left: 1, 2, ... (2x2 is 1 2 / 3 4)
  --once               print one frame and exit
  --help               this text
  --version            print the version

Keys (not case-sensitive)
  Tab, n next scene    space read now    s read interval    p / m CPU / memory    1, c split by core
  z skin    i status line    t switch to top now    h, ? help    q, Ctrl+C quit

More: man tidetop
`

func fail(msg string) {
	fmt.Fprintf(os.Stderr, "tidetop: %s\n", msg)
	os.Exit(1)
}

type gridOpt struct{ across, down int }

type options struct {
	grid               *gridOpt
	gridText           string
	pos                float64
	posSet             bool
	mem, top           bool
	topBelow, topAfter float64
	scene, art, skin   string
	sceneSet, demo     bool
	data, quality      string
	fps                float64
	fpsSet, once, help bool
}

func parseArgs(argv []string) options {
	o := options{topBelow: 20, topAfter: 5, scene: defaultScene, pos: math.NaN()}
	for i := 0; i < len(argv); i++ {
		flag, inline, hasInline := strings.Cut(argv[i], "=")
		value := func() string {
			if hasInline {
				return inline
			}
			if i+1 < len(argv) {
				i++
				return argv[i]
			}
			fail(flag + " 에 값이 없다")
			return ""
		}
		num := func() float64 {
			v, err := strconv.ParseFloat(value(), 64)
			if err != nil {
				return math.NaN()
			}
			return v
		}
		switch flag {
		case "--help", "-h":
			o.help = true
		case "--version", "-v":
			fmt.Println("tidetop " + version)
			os.Exit(0)
		case "--once":
			o.once = true
		case "--demo":
			o.demo = true
		case "-t", "--top":
			o.top = true
		case "-M", "--mem":
			o.mem = true
		case "--grid":
			o.gridText = value()
		case "--pos":
			o.pos, o.posSet = num(), true
		case "-m", "--top-below":
			o.topBelow = num()
		case "-d", "--top-after":
			o.topAfter = num()
		case "--art":
			o.art = value()
		case "--scene":
			o.scene, o.sceneSet = value(), true
		case "--data":
			o.data = value()
		case "--skin":
			o.skin = value()
		case "--quality":
			o.quality = value()
		case "--fps":
			o.fps, o.fpsSet = num(), true
		default:
			fail("모르는 옵션: " + argv[i] + " (--help 를 본다)")
		}
	}
	if o.gridText != "" || o.posSet {
		m := regexp.MustCompile(`(?i)^(\d+)x(\d+)$`).FindStringSubmatch(o.gridText)
		if m == nil {
			fail("--grid 는 <가로>x<세로> 다. 예: --grid 2x2")
		}
		a, _ := strconv.Atoi(m[1])
		d, _ := strconv.Atoi(m[2])
		o.grid = &gridOpt{a, d}
		if a < 1 || d < 1 || a*d > 64 {
			fail("--grid 의 조각은 1~64 개다")
		}
		if !(o.pos == math.Trunc(o.pos) && o.pos >= 1 && o.pos <= float64(a*d)) {
			fail(fmt.Sprintf("--pos 는 1~%d 이다", a*d))
		}
		if o.art != "" || (o.scene != "grove" && o.scene != "stars" && o.scene != "library") {
			fail("--grid 는 grove, stars, library 에서만 쓴다")
		}
	}
	if o.mem && o.data != "" {
		fail("-M 은 이 컴퓨터의 값을 볼 때만 쓴다 (--data 와 같이 쓰지 않는다)")
	}
	if o.top && o.data != "" {
		fail("-t 는 이 컴퓨터의 값을 볼 때만 쓴다 (--data 와 같이 쓰지 않는다)")
	}
	if !(o.topBelow >= 0 && o.topBelow <= 100) {
		fail("-m 은 0~100 이다")
	}
	if !(o.topAfter >= 0) {
		fail("-d 는 0 이상의 초다")
	}
	if o.demo && o.data != "" {
		fail("--demo 와 --data 는 같이 쓰지 않는다")
	}
	if o.art != "" && o.sceneSet {
		fail("--art 와 --scene 은 같이 쓰지 않는다")
	}
	if o.art != "" && o.skin != "" {
		fail("--art 에는 스킨이 없다")
	}
	s, ok := registry[o.scene]
	if !ok {
		fail(fmt.Sprintf("모르는 장면: %s (%s)", o.scene, strings.Join(sceneNames, " | ")))
	}
	if o.skin != "" && !slices.Contains(s.Skins(), o.skin) {
		if len(s.Skins()) > 0 {
			fail(fmt.Sprintf("%s 의 스킨은 %s 이다", o.scene, strings.Join(s.Skins(), ", ")))
		}
		fail(o.scene + " 에는 스킨이 없다")
	}
	if o.quality != "" && o.quality != "low" && o.quality != "high" {
		fail("--quality 는 low 나 high 다")
	}
	if o.fpsSet && !(o.fps >= 1 && o.fps <= 60) {
		fail("--fps 는 1~60 이다")
	}
	return o
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if len(os.Args) < 3 {
			if korean {
				fail("init 에는 폴더가 있어야 한다")
			}
			fail("init needs a folder")
		}
		lang := "en"
		if korean {
			lang = "ko"
		}
		msg, err := art.MakeSamples(os.Args[2], lang)
		if err != nil {
			fail(err.Error())
		}
		fmt.Print(msg)
		return
	}
	o := parseArgs(os.Args[1:])
	if o.help {
		help := helpEn
		if korean {
			help = helpKo
		}
		fmt.Print(strings.NewReplacer("{{scenes}}", strings.Join(sceneNames, " | "), "{{default}}", defaultScene).Replace(help))
		return
	}
	run(o)
}

type app struct {
	o        options
	name     string // 지금 장면의 이름
	scene    render.Scene
	src      source.Source
	quality  string
	color    bool
	fps      float64
	remote   bool
	asked    time.Time // 터미널에게 다 그렸는지 물은 때. 답을 받으면 0 으로 돌린다
	answered bool      // 터미널이 한 번이라도 답했는가
	replies  chan struct{}
	skin     string
	target   []feed.Computed
	shown    []feed.Computed
	stale    bool
	oldFor   time.Duration // 파일 피드가 그대로 있은 시간(끊김일 때). 메인 고루틴에서만 읽고 쓴다
	problem  string
	chosen   time.Duration
	polledAt time.Time
	busy     bool
	note     *note
	statusOn bool
	still    int
	screen   *term.Screen
	restore  func()
	below    time.Time
}

type note struct {
	lines func() []string
	until time.Time
	help  bool
}

func run(o options) {
	out := os.Stdout
	a := &app{o: o, scene: registry[o.scene], skin: o.skin, name: o.scene}
	lang := "en"
	if korean {
		lang = "ko"
	}
	if o.art != "" {
		s, err := art.Load(o.art, lang)
		if err != nil {
			fail(err.Error())
		}
		a.scene = s
	}
	if o.grid != nil {
		p := int(o.pos) - 1
		a.scene = render.Crop{Scene: a.scene, Across: o.grid.across, Down: o.grid.down, Col: p % o.grid.across, Row: p / o.grid.across}
	}
	found := term.Detect(out)
	a.quality = o.quality
	if a.quality == "" {
		a.quality = found.Quality
	}
	a.color = a.quality == "high" || found.Color
	a.fps = o.fps
	a.remote = found.Remote
	if !o.fpsSet {
		a.fps = 2
		if a.quality == "high" {
			a.fps = 15
			if a.remote {
				a.fps = 8 // 화면이 네트워크를 건너가면 덜 보낸다
			}
		}
	}
	switch {
	case o.data != "":
		d, err := source.NewData(o.data)
		if err != nil {
			fail(err.Error())
		}
		a.src = d
	case o.demo:
		started := time.Now()
		if o.grid != nil {
			started = time.Unix(0, 0)
		}
		a.src = source.NewDemo(a.scene.Local(), started)
	default:
		a.src = source.NewLocal(a.scene.Local())
	}
	if o.mem {
		sw, ok := a.switchable()
		if !ok {
			name := o.scene
			if o.art != "" {
				name = "--art"
			}
			fail(name + " 에는 메모리 기준이 없다")
		}
		sw.SetBasis("mem")
	}
	ctx := context.Background()
	a.poll(ctx)
	if a.target == nil {
		fail(a.problem)
	}
	if o.once || !term.IsTerminal(out) {
		a.shown = a.target
		cols, rows := term.Size(out)
		if term.IsTerminal(out) {
			rows-- // 터미널에서는 마지막 줄바꿈이 한 줄을 밀어 올리므로 한 줄을 덜 그린다
		}
		fmt.Println(strings.Join(render.Lines(a.frame(0, cols, rows)), "\n"))
		return
	}
	a.loop(ctx, out)
}

func (a *app) switchable() (source.Switchable, bool) {
	sw, ok := a.src.(source.Switchable)
	local := a.scene.Local()
	return sw, ok && (local == "cpu" || local == "free")
}

func (a *app) poll(ctx context.Context) {
	r := a.src.Next(ctx)
	a.apply(r)
}

func (a *app) apply(r feed.Result) {
	a.oldFor = 0
	if r.OK {
		a.target = feed.Compute(r.Feed)
		// 파일 피드가 오래 그대로면 값은 받되 끊김으로 덮는다
		a.stale = r.Code == "old"
		if d, ok := a.src.(*source.Data); ok && a.stale {
			a.oldFor = d.Old() // Next 가 결과를 보낸 뒤라 읽기와 겹치지 않는다
		}
	} else {
		a.problem = r.Error
		a.stale = true
	}
}

func (a *app) frame(t float64, cols, rows int) *render.Grid {
	state := "live"
	if a.stale {
		state = "stale"
	}
	in := render.Input{Values: a.shown, State: state, Cols: cols, Rows: rows, Time: t, Quality: a.quality, Color: a.color, Skin: a.skin}
	if sw, ok := a.src.(source.Switchable); ok && sw.Split() && len(a.shown) > 1 {
		return render.RenderTiles(a.scene, in)
	}
	return render.Render(a.scene, in)
}

func (a *app) interval() time.Duration {
	d := a.src.Every()
	if a.chosen > 0 {
		d = a.chosen
	}
	return max(d, a.src.Floor())
}

// 쓰는 정도(%)와 여유. 여유를 읽는 장면은 받은 값이 여유다. 물결은 둘째 값(CPU)을 본다.
func (a *app) raws() []float64 {
	out := make([]float64, len(a.target))
	for i, v := range a.target {
		out[i] = v.Raw.Value
	}
	return out
}

func average(l []float64) float64 {
	s := 0.0
	for _, x := range l {
		s += x
	}
	return s / math.Max(1, float64(len(l)))
}

func (a *app) used() float64 {
	r := a.raws()
	switch a.scene.Local() {
	case "mem-cpu":
		if len(r) > 1 {
			return r[1]
		}
		return 0
	case "free":
		return 100 - average(r)
	}
	return average(r)
}

func (a *app) headroom() float64 { return 100 - a.used() }

func pct(x float64) string { return fmt.Sprintf("%d%%", int(math.Round(x))) }

func secs(d time.Duration) string { return strconv.FormatFloat(d.Seconds(), 'f', -1, 64) }

func (a *app) basis() string {
	if sw, ok := a.src.(source.Switchable); ok {
		return sw.Basis()
	}
	return "cpu"
}

func (a *app) status() string {
	name := a.name
	if a.o.art != "" {
		name = "art"
	} else if a.skin != "" {
		name += " " + a.skin
	}
	if a.o.grid != nil {
		name += fmt.Sprintf(" %d/%d", int(a.o.pos), a.o.grid.across*a.o.grid.down)
	}
	if a.o.data != "" {
		v := "-"
		if len(a.target) > 0 {
			v = strconv.FormatFloat(a.target[0].Raw.Value, 'f', -1, 64)
		}
		label := "value"
		if korean {
			label = "값"
		}
		if a.stale && a.oldFor > 0 {
			secs := int(a.oldFor.Seconds())
			name += " | " + a.say(fmt.Sprintf("파일이 %d초째 그대로", secs), fmt.Sprintf("file unchanged for %ds", secs))
		}
		return fmt.Sprintf("[%s %s | %s]", label, v, name)
	}
	if a.scene.Local() == "mem-cpu" {
		r := a.raws()
		m := 0.0
		if len(r) > 0 {
			m = r[0]
		}
		return fmt.Sprintf("[MEM %s CPU %s | %s]", pct(m), pct(a.used()), name)
	}
	many := ""
	if len(a.target) > 1 {
		many = fmt.Sprintf(" ×%d", len(a.target))
	}
	b := "CPU"
	if a.basis() == "mem" {
		b = "MEM"
	}
	return fmt.Sprintf("[%s %s%s | %s]", b, pct(a.used()), many, name)
}

func (a *app) show(lines func() []string, seconds float64) {
	a.note = &note{lines: lines, until: time.Now().Add(time.Duration(seconds * float64(time.Second)))}
	a.still = 0
}

func fixed(lines ...string) func() []string { return func() []string { return lines } }

func (a *app) say(ko, en string) string {
	if korean {
		return ko
	}
	return en
}

func (a *app) helpLines() []string {
	skin := a.say(" (없음)", " (none)")
	if s := a.scene.Skins(); len(s) > 0 {
		now := a.skin
		if now == "" {
			now = s[0]
		}
		skin = a.say(" (지금 "+now+")", " (now "+now+")")
	}
	basis := a.say("CPU", "CPU")
	if a.basis() == "mem" {
		basis = a.say("메모리", "memory")
	}
	title := a.name
	if a.o.art != "" {
		title = "art"
	}
	if korean {
		kt := sceneTitles[a.name]
		if a.o.art != "" {
			kt = "글자 그림"
		}
		return []string{
			"Tab n     장면 바꾸기 (지금 " + kt + ")",
			"스페이스  지금 읽기",
			"s         읽는 주기 (지금 " + secs(a.interval()) + "초)",
			"p / m     CPU / 메모리 기준 (지금 " + basis + ")",
			"1 / c     코어마다 나눠 보기",
			"z         스킨" + skin,
			"i         상태 줄",
			"t         top 으로 넘어가기 (돌아오지 않음)",
			"h ?       도움말 닫기",
			"q         끝내기",
			"대소문자를 가리지 않는다",
		}
	}
	return []string{
		"Tab n  next scene (now " + title + ")",
		"space  read now",
		"s      read interval (now " + secs(a.interval()) + "s)",
		"p / m  CPU / memory (now " + basis + ")",
		"1 / c  split by core",
		"z      skin" + skin,
		"i      status line",
		"t      switch to top (does not come back)",
		"h ?    close help",
		"q      quit",
		"Keys are not case-sensitive",
	}
}

var steps = []float64{0.5, 1, 2, 5}     // 이 컴퓨터, 흉내 값, 파일
var remoteSteps = []float64{10, 30, 60} // https 주소

// 키 하나를 처리한다. 끝낼 때는 false 를 돌려준다.
func (a *app) press(code byte, readNow func()) bool {
	if code == 3 {
		return false // Ctrl+C
	}
	key := code
	if key >= 'A' && key <= 'Z' {
		key += 'a' - 'A' // 대소문자를 가리지 않는다
	}
	sw, canSwitch := a.switchable()
	canSwitch = canSwitch && a.o.data == ""
	fixedMsg := a.say("이 장면은 값을 여럿 읽어 바꿀 수 없음", "this scene reads several values and cannot be switched")
	if a.o.data != "" {
		fixedMsg = a.say("다른 곳의 피드는 바꿀 수 없음", "a feed from elsewhere cannot be switched")
	}
	switch key {
	case 'q':
		return false
	case '\t', 'n':
		// 다음 장면. 글자 그림과 이어 붙인 창에서는 바꾸지 않는다
		if a.o.art != "" || a.o.grid != nil {
			a.show(fixed(a.say("글자 그림(--art)에서는 장면을 바꿀 수 없음", "the scene cannot be changed with --art")), 2)
			if a.o.grid != nil {
				a.show(fixed(a.say("이어 붙인 창(--grid)에서는 장면을 바꿀 수 없음", "the scene cannot be changed in a joined window (--grid)")), 2)
			}
			return true
		}
		a.name = sceneOrder[(slices.Index(sceneOrder, a.name)+1)%len(sceneOrder)]
		a.scene = registry[a.name]
		a.skin = ""
		switch s := a.src.(type) {
		case *source.Local:
			s.Kind = a.scene.Local()
		case *source.Demo:
			s.Kind = a.scene.Local()
		}
		// 값을 여럿 읽는 장면에는 코어 나눔과 메모리 기준이 없다
		if sw, ok := a.src.(source.Switchable); ok {
			if _, one := a.switchable(); !one {
				sw.SetSplit(false)
				sw.SetBasis("cpu")
			}
		}
		// https 피드는 바닥 간격 안에 다시 묻지 않는다. 받은 값은 장면과 상관없어 그대로 그린다
		if time.Since(a.polledAt) >= a.src.Floor() {
			readNow()
		}
		a.show(fixed(a.say("장면 "+sceneTitles[a.name]+" ("+a.name+")", "scene "+a.name)), 2)
	case ' ':
		if wait := a.src.Floor() - time.Since(a.polledAt); wait > 0 {
			s := secs(a.src.Floor())
			a.show(fixed(a.say(s+"초 안에는 다시 묻지 않음", "not asking again within "+s+"s")), 2)
			return true
		}
		readNow()
		a.show(fixed(a.say("지금 읽음", "read now")), 2)
	case 's':
		floor := a.src.Floor().Seconds()
		list := steps
		if floor >= 10 {
			list = []float64{floor}
			for _, x := range remoteSteps {
				if x > floor {
					list = append(list, x)
				}
			}
		}
		now := a.interval().Seconds()
		next := list[0]
		for _, x := range list {
			if x > now {
				next = x
				break
			}
		}
		a.chosen = time.Duration(next * float64(time.Second))
		s := secs(a.interval())
		a.show(fixed(a.say(s+"초마다 읽음", "reading every "+s+"s")), 2)
	case 'p', 'm':
		if !canSwitch {
			a.show(fixed(fixedMsg), 2)
			return true
		}
		if key == 'm' {
			sw.SetBasis("mem")
		} else {
			sw.SetBasis("cpu")
		}
		sw.SetSplit(false)
		readNow()
		a.show(func() []string {
			b := a.say("CPU", "CPU")
			if a.basis() == "mem" {
				b = a.say("메모리", "memory")
			}
			return []string{a.say(
				fmt.Sprintf("%s 기준 (사용 %s, 여유 %s)", b, pct(a.used()), pct(a.headroom())),
				fmt.Sprintf("%s (used %s, headroom %s)", b, pct(a.used()), pct(a.headroom())))}
		}, 2)
	case '1', 'c':
		if !canSwitch {
			a.show(fixed(fixedMsg), 2)
			return true
		}
		if a.o.grid != nil {
			a.show(fixed(a.say("이어 붙인 창(--grid)에서는 나눠 볼 수 없음", "a joined window (--grid) cannot be split")), 2)
			return true
		}
		if sw.Split() {
			sw.SetSplit(false)
			readNow()
			a.show(fixed(a.say("나누지 않고 봄", "showing the whole")), 2)
			return true
		}
		n := min(runtime.NumCPU(), 64)
		if a.o.demo {
			n = 8
		}
		cols, rows := term.Size(os.Stdout)
		if _, ok := render.TileLayout(n, cols, rows); !ok {
			a.show(fixed(a.say(fmt.Sprintf("창이 작아 코어 %d개로 나눌 수 없음", n), fmt.Sprintf("the window is too small for %d cores", n))), 2)
			return true
		}
		fromMem := sw.Basis() == "mem"
		sw.SetBasis("cpu")
		sw.SetSplit(true)
		readNow()
		msg := a.say(fmt.Sprintf("CPU 코어 %d개로 나눠 봄", n), fmt.Sprintf("split into %d CPU cores", n))
		if fromMem {
			msg = a.say(fmt.Sprintf("메모리에서 CPU 로 바꿔 코어 %d개로 나눠 봄", n), fmt.Sprintf("switched from memory to CPU, split into %d cores", n))
		}
		a.show(fixed(msg), 2)
	case 'z':
		skins := a.scene.Skins()
		if len(skins) == 0 {
			a.show(fixed(a.say("이 장면에는 스킨이 없음", "this scene has no skins")), 2)
			return true
		}
		now := a.skin
		if now == "" {
			now = skins[0]
		}
		a.skin = skins[(slices.Index(skins, now)+1)%len(skins)]
		a.show(fixed(a.say("스킨 "+a.skin, "skin "+a.skin)), 2)
	case 'i':
		a.statusOn = !a.statusOn
		a.still = 0
	case 't':
		a.switchToTop(a.headroom(), true)
	case 'h', '?':
		if a.note != nil && a.note.help {
			a.note = nil
			a.still = 0
			return true
		}
		a.show(a.helpLines, 8)
		a.note.help = true
	}
	return true
}

// waiting: 앞 장에 대한 터미널의 답을 기다리는 중인가.
// 첫 물음에 1초 안에 답하지 않은 터미널은 묻지 않고 그린다. 답한 적이 있으면 길이 느려도 답을 기다린다
// (30초가 지나면 답이 사라졌다고 보고 다시 그린다).
func (a *app) waiting(now time.Time) bool {
	if a.asked.IsZero() {
		return false
	}
	limit := time.Second
	if a.answered {
		limit = 30 * time.Second
	}
	if now.Sub(a.asked) < limit {
		return true
	}
	a.asked = time.Time{}
	return false
}

func (a *app) quit() {
	if a.screen != nil {
		a.screen.Close()
	}
	// 아직 오지 않은 답이 끝난 뒤 셸에 글자로 찍히지 않게 잠깐 받아 버린다
	if !a.asked.IsZero() && a.replies != nil {
		select {
		case <-a.replies:
		case <-time.After(500 * time.Millisecond):
		}
		a.asked = time.Time{}
	}
	if a.restore != nil {
		a.restore()
	}
}

// top 으로 넘어간다. 한 번 넘어가면 돌아오지 않는다. 맥과 리눅스에서는 tidetop 이 top 으로 바뀌므로 tidetop 은 아무것도 남기지 않는다.
func (a *app) switchToTop(h float64, byKey bool) {
	path, err := exec.LookPath("top")
	if runtime.GOOS == "windows" || err != nil {
		a.show(fixed(a.say("이 컴퓨터에는 top 이 없음", "top is not available on this computer")), 3)
		return
	}
	a.quit()
	if byKey {
		fmt.Println(a.say("tidetop: t 키로 top 으로 넘어간다", "tidetop: switching to top (key t)"))
	} else {
		fmt.Println(a.say(
			fmt.Sprintf("tidetop: 여유 %d%% (문턱 %s%%) 가 %s초 이어져 top 으로 넘어간다", int(math.Round(h)), strconv.FormatFloat(a.o.topBelow, 'f', -1, 64), strconv.FormatFloat(a.o.topAfter, 'f', -1, 64)),
			fmt.Sprintf("tidetop: headroom %d%% (line %s%%) lasted %ss, switching to top", int(math.Round(h)), strconv.FormatFloat(a.o.topBelow, 'f', -1, 64), strconv.FormatFloat(a.o.topAfter, 'f', -1, 64))))
	}
	if err := syscall.Exec(path, []string{"top"}, os.Environ()); err != nil {
		fail("top: " + err.Error())
	}
}

// -t: 보고 있는 여유(%)가 문턱 이하로 정한 시간 동안 이어지면 top 으로 넘어간다
func (a *app) watchTop() {
	if !a.o.top || a.target == nil {
		return
	}
	h := a.headroom()
	if h > a.o.topBelow {
		a.below = time.Time{}
		return
	}
	if a.below.IsZero() {
		a.below = time.Now()
	}
	if time.Since(a.below) >= time.Duration(a.o.topAfter*float64(time.Second)) {
		a.switchToTop(h, false)
	}
}

func (a *app) loop(ctx context.Context, out *os.File) {
	a.screen = term.NewScreen(out)
	// 색이 한 단계만 바뀐 칸은 다시 쓰지 않는다. SSH 너머에서는 두 단계까지
	a.screen.Tol = 4
	if a.remote {
		a.screen.Tol = 8
	}
	restore, _ := term.Raw(os.Stdin)
	a.restore = restore
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	keys := make(chan byte, 16)
	replies := make(chan struct{}, 4)
	go func() {
		buf := make([]byte, 64)
		var in term.Input
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			for _, b := range buf[:n] {
				k, isKey, reply := in.Feed(b)
				if reply {
					replies <- struct{}{}
				} else if isKey {
					keys <- k
				}
			}
		}
	}()
	a.replies = replies
	results := make(chan feed.Result, 1)
	pollTimer := time.NewTimer(a.interval())
	a.polledAt = time.Now()
	readNow := func() {
		if a.busy {
			return
		}
		a.busy = true
		pollTimer.Stop()
		a.polledAt = time.Now()
		go func() { results <- a.src.Next(ctx) }()
	}
	a.screen.Open()
	// 그리기 전에 한 번 물어 이 터미널이 답하는지 본다. 이 답은 길을 한 번 오가는 시간만큼만 걸린다
	io.WriteString(out, term.Ask)
	a.asked = time.Now()
	defer a.quit()
	a.watchTop()

	started := time.Now()
	clock := func(now time.Time) float64 { return now.Sub(started).Seconds() }
	if a.o.grid != nil {
		// 이어 붙인 창은 벽시계(하루 단위)를 써서 다른 컴퓨터의 조각과 움직임이 이어진다
		clock = func(now time.Time) float64 { return math.Mod(float64(now.UnixMilli())/1000, 86400) }
	}
	last := started
	lastCols, lastRows := 0, 0
	tick := time.NewTimer(0)
	for {
		select {
		case <-signals:
			return
		case k := <-keys:
			if !a.press(k, readNow) {
				return
			}
		case <-replies:
			a.asked = time.Time{}
			a.answered = true
		case <-pollTimer.C:
			readNow()
		case r := <-results:
			a.apply(r)
			a.busy = false
			a.watchTop()
			pollTimer.Reset(a.interval())
		case now := <-tick.C:
			if a.waiting(now) {
				// 앞 장이 아직 터미널에 다 그려지지 않았다. 더 보내지 않는다
				tick.Reset(time.Duration(1000/a.fps) * time.Millisecond)
				continue
			}
			a.shown = source.Ease(a.shown, a.target, now.Sub(last).Seconds(), 0.9)
			last = now
			cols, rows := term.Size(out)
			if cols != lastCols || rows != lastRows {
				a.screen.Clear()
				lastCols, lastRows = cols, rows
			}
			g := a.frame(clock(now), cols, rows)
			if a.note != nil && now.After(a.note.until) {
				a.note = nil
			}
			if a.note != nil {
				render.Stamp(g, a.note.lines(), a.quality == "high", a.color, "")
			}
			if a.statusOn {
				render.Stamp(g, []string{a.status()}, a.quality == "high", a.color, "bottom-right")
			}
			drawn := time.Now()
			if a.screen.Draw(g) {
				a.still = 0
				if a.answered {
					io.WriteString(out, term.Ask)
					a.asked = time.Now()
				}
			} else {
				a.still++
			}
			took := time.Since(drawn)
			// 화면이 멈춰 있는 동안에는 1초에 네 번만 들여다본다
			wait := time.Duration(1000/a.fps) * time.Millisecond
			if float64(a.still) > a.fps/2 {
				wait = max(250*time.Millisecond, wait)
			}
			// 쓰는 데 오래 걸렸으면 터미널이나 네트워크가 밀린 것이다. 그 두 배는 쉰다
			wait = max(wait, min(2*took, 2*time.Second))
			tick.Reset(wait)
		}
	}
}
