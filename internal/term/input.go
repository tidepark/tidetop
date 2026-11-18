package term

// 터미널에게 「여기까지 그렸으면 알려 달라」고 묻는다(커서 자리 묻기, ESC[6n).
// 터미널은 앞서 받은 것을 다 그린 뒤에 ESC[줄;칸R 로 답한다. 이 답을 받기 전에는 다음 장을 보내지 않으면
// SSH 같은 긴 길에 화면이 쌓이지 않고, 키가 늦게 먹지 않는다.
const Ask = esc + "6n"

// Input 은 키보드에서 온 바이트를 키와 터미널의 답으로 가른다.
// 화살표 같은 ESC[ 로 시작하는 부호는 키로 넘기지 않는다.
type Input struct {
	state  int // 0 보통, 1 ESC 를 받음, 2 ESC[ 안
	params []byte
}

// Feed 는 바이트 하나를 받는다. 키면 key 와 true, 터미널의 답이면 reply 가 true 다.
// ESC 하나만 오고 다음 바이트가 [ 가 아니면 ESC 는 버리고 다음 바이트를 키로 본다.
func (in *Input) Feed(b byte) (key byte, isKey, reply bool) {
	switch in.state {
	case 1:
		if b == '[' {
			in.state, in.params = 2, in.params[:0]
			return 0, false, false
		}
		in.state = 0
		if b == 0x1b {
			in.state = 1
			return 0, false, false
		}
		return b, true, false
	case 2:
		if b >= 0x20 && b <= 0x3f {
			if len(in.params) < 16 {
				in.params = append(in.params, b)
			}
			return 0, false, false
		}
		in.state = 0
		return 0, false, b == 'R' && isPosition(in.params)
	}
	if b == 0x1b {
		in.state = 1
		return 0, false, false
	}
	return b, true, false
}

// 줄;칸 모양인가 (숫자;숫자)
func isPosition(p []byte) bool {
	semi := 0
	for i, c := range p {
		switch {
		case c == ';':
			if i == 0 || i == len(p)-1 {
				return false
			}
			semi++
		case c < '0' || c > '9':
			return false
		}
	}
	return semi == 1
}
