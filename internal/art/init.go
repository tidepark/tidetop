package art

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tidepark/tidetop/internal/feed"
	"github.com/tidepark/tidetop/internal/render"
	"github.com/tidepark/tidetop/internal/scenes"
)

// MakeSamples 는 tidetop init <폴더> 다. 글자 그림 폴더의 견본을 만든다.
// 모닥불을 ASCII 로 뽑은 0.txt ~ 100.txt 열한 장과 쓰는 법(README.txt)을 둔다.
// 폴더가 이미 있고 비어 있지 않으면 만들지 않는다. 덮어쓰지 않는다.
func MakeSamples(dir, lang string) (string, error) {
	say := initMessages[lang]
	if say == nil {
		say = initMessages["ko"]
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return "", fmt.Errorf(say["busy"], dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for v := 0; v <= 100; v += 10 {
		if err := writeNew(filepath.Join(dir, fmt.Sprintf("%d.txt", v)), picture(float64(v))); err != nil {
			return "", err
		}
	}
	if err := writeNew(filepath.Join(dir, "README.txt"), sampleReadme); err != nil {
		return "", err
	}
	return fmt.Sprintf(say["done"], dir, dir), nil
}

func writeNew(path, text string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

var sgrCodes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// 견본 한 장: 모닥불을 low(ASCII) 로 세 시각에 찍는다. 같은 장이 이어지면 하나만 둔다(값 0 의 불씨는 멈춰 있다).
func picture(value float64) string {
	r := feed.Validate(map[string]any{"spec": 1.0, "values": []any{map[string]any{"value": value, "to": 100.0}}})
	values := feed.Compute(r.Feed)
	var frames []string
	seen := map[string]bool{}
	for _, t := range []float64{0, 0.45, 0.9} {
		g := render.Render(scenes.Campfire{}, render.Input{Values: values, State: "live", Cols: 40, Rows: 12, Time: t, Quality: "low", Color: false})
		var lines []string
		for _, l := range render.Lines(g) {
			lines = append(lines, strings.TrimRight(sgrCodes.ReplaceAllString(l, ""), " \t"))
		}
		text := strings.Join(lines, "\n")
		if !seen[text] {
			seen[text] = true
			frames = append(frames, text)
		}
	}
	return strings.Join(frames, "\n---\n") + "\n"
}

var initMessages = map[string]map[string]string{
	"ko": {"busy": "폴더가 비어 있지 않아 만들지 않았다: %s", "done": "견본을 만들었다: %s\n띄우기: tidetop --art %s\n"},
	"en": {"busy": "the folder is not empty, nothing was written: %s", "done": "samples written: %s\nshow them: tidetop --art %s\n"},
}

const sampleReadme = `tidetop 글자 그림 폴더 / tidetop text picture folder

[한국어]
이 폴더를 장면으로 띄운다:
  tidetop --art <이 폴더>

- <숫자>.txt 가 그림이다. 값이 그 숫자 이상이면 그 그림을 보여 준다.
  값이 가장 작은 숫자보다 작으면 가장 작은 그림을 쓴다.
- 숫자는 값의 단위 그대로다. --data 없이 띄우면 이 컴퓨터의 CPU 사용률(%)이다.
  0~10 인 피드면 0.txt~10.txt, 켜짐·꺼짐이면 0.txt 와 1.txt 만 두면 된다.
- 몇 장만 있어도 된다. 필요 없는 파일은 지우고, 사이 숫자를 더해도 된다.
- 한 파일에 '---' 만 있는 줄로 여러 장을 나누면 1초에 두 장씩 넘긴다.
- 그림은 화면 가운데에 놓인다. 넘치면 잘린다.
- 색 부호 같은 터미널 제어 글자는 걸러 낸다. 글자만 보인다.
- 견본은 모닥불이다. 0 은 불씨, 100 은 큰 불이다.

[English]
Show this folder as the scene:
  tidetop --art <this folder>

- Each <number>.txt is a picture. A picture shows when the value is at least its number.
  Below the smallest number, the smallest picture shows.
- Numbers use the value's own units. Without --data, the value is this computer's CPU use (%).
  For a 0-10 feed use 0.txt to 10.txt; for on/off, 0.txt and 1.txt are enough.
- A few pictures are enough. Delete files you do not need, or add numbers in between.
- Split a file into frames with a line holding only '---'. Frames change twice a second.
- Pictures are centred on the screen and cut where they do not fit.
- Terminal control codes such as colour codes are removed. Only the characters show.
- The samples are a campfire. 0 is embers, 100 is a large fire.
`
