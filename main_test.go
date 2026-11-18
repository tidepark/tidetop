package main

import (
	"context"
	"testing"
	"time"

	"github.com/tidepark/tidetop/internal/feed"
)

// floorSource 는 https 피드처럼 10초보다 자주 묻지 않는 출처다.
type floorSource struct{}

func (floorSource) Next(context.Context) feed.Result { return feed.Result{} }
func (floorSource) Every() time.Duration             { return 10 * time.Second }
func (floorSource) Floor() time.Duration             { return 10 * time.Second }

// 장면을 바꿔도 https 피드를 10초 안에 다시 묻지 않는다.
func TestTabKeepsFloor(t *testing.T) {
	a := &app{o: options{data: "https://example.com/feed.json"}, name: "grove", scene: registry["grove"], src: floorSource{}, polledAt: time.Now()}
	reads := 0
	for range 6 {
		a.press('\t', func() { reads++ })
	}
	if reads != 0 {
		t.Fatalf("10초 안에 장면을 바꾸며 %d번 물었다", reads)
	}
	if a.name != "grove" {
		t.Fatalf("장면이 여섯 번 돌아 grove 로 와야 하는데 %s 다", a.name)
	}
	a.polledAt = time.Now().Add(-11 * time.Second)
	a.press('\t', func() { reads++ })
	if reads != 1 {
		t.Fatalf("10초가 지난 뒤 장면을 바꾸면 한 번 물어야 하는데 %d번 물었다", reads)
	}
}
