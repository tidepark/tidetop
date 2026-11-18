//go:build darwin

package source

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/shirou/gopsutil/v4/mem"
)

var pageSize = regexp.MustCompile(`page size of (\d+)`)

// memoryUsed 는 쓰는 메모리의 비율이다. 맥은 남는 메모리를 캐시로 채워 두므로,
// vm_stat 의 활성 + 고정 + 압축을 쓰는 것으로 센다(활성 상태 보기의 「사용된 메모리」에 가깝다).
func memoryUsed(ctx context.Context) (float64, bool) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil || v.Total == 0 {
		return 0, false
	}
	out, err := exec.CommandContext(ctx, "vm_stat").Output()
	if err != nil {
		return v.UsedPercent / 100, true
	}
	text := string(out)
	page := 16384.0
	if m := pageSize.FindStringSubmatch(text); m != nil {
		page, _ = strconv.ParseFloat(m[1], 64)
	}
	pages := func(name string) float64 {
		m := regexp.MustCompile(name + `:\s+(\d+)`).FindStringSubmatch(text)
		if m == nil {
			return 0
		}
		n, _ := strconv.ParseFloat(m[1], 64)
		return n
	}
	used := pages("Pages active") + pages("Pages wired down") + pages("Pages occupied by compressor")
	if used <= 0 {
		return v.UsedPercent / 100, true
	}
	return used * page / float64(v.Total), true
}
