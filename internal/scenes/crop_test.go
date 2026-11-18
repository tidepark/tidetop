package scenes

import (
	"testing"

	"github.com/tidepark/tidetop/internal/feed"
	"github.com/tidepark/tidetop/internal/render"
)

// 도서관을 이어 붙인 창의 조각으로 그리면, 큰 그림을 한 번에 그려 그 자리를 잘라 낸 것과 같아야 한다.
// (주의 줄은 조각마다 따로 뽑으므로 여유가 10% 를 넘는 값만 견준다.)
func TestLibraryCropMatchesWhole(t *testing.T) {
	const across, down, cols, rows = 3, 2, 40, 14
	for _, v := range []float64{35, 70, 100, 130} {
		for _, q := range []string{"high", "low"} {
			for _, tm := range []float64{3, 47.5} {
				r := feed.Validate(map[string]any{"spec": 1.0, "values": []any{map[string]any{"value": v, "to": 100.0}}})
				if !r.OK {
					t.Fatal(r.Error)
				}
				in := render.Input{Values: feed.Compute(r.Feed), State: "live", Time: tm, Quality: q, Color: true}
				whole := in
				whole.Cols, whole.Rows = cols*across, rows*down
				big := render.Render(Library{}, whole)
				for row := 0; row < down; row++ {
					for col := 0; col < across; col++ {
						piece := in
						piece.Cols, piece.Rows = cols, rows
						g := render.Render(render.Crop{Scene: Library{}, Across: across, Down: down, Col: col, Row: row}, piece)
						for y := 0; y < rows; y++ {
							for x := 0; x < cols; x++ {
								i, j := y*cols+x, (row*rows+y)*big.Cols+col*cols+x
								if g.Ch[i] != big.Ch[j] || g.Fg[i] != big.Fg[j] || g.Bg[i] != big.Bg[j] {
									t.Fatalf("값 %v, %s, %v초, 조각 (%d,%d) 의 칸 (%d,%d) 가 큰 그림과 다르다", v, q, tm, col, row, x, y)
								}
							}
						}
					}
				}
			}
		}
	}
}
