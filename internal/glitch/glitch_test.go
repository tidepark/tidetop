package glitch

import (
	"testing"

	"github.com/tidepark/tidetop/internal/noise"
)

// 잠깐 범위 아래로 내려갔다 올라온 뒤, 부드럽게 따라가며 남는 아주 작은 값으로는 줄이 서지 않는다
func TestNoLingeringLine(t *testing.T) {
	under := 0.1
	k := 1 - 0.07 // 초당 15장, 0.9초로 따라갈 때 한 장마다 남는 몫
	for frame := 0; frame < 15*10; frame++ {
		under *= k
	}
	if under <= 0 {
		t.Fatal("따라가는 값은 0 이 되지 않는다(이 테스트의 전제)")
	}
	if lines := Make(noise.Smooth(0, 0.1, under), 120, 80, 3, 0); lines != nil {
		t.Errorf("10초 뒤에도 줄 %d 개", len(lines))
	}
	if Make(1, 120, 80, 3, 0) == nil && Make(1, 120, 80, 3.4, 0) == nil {
		t.Error("범위 아래에서는 줄이 선다")
	}
}
