//go:build !darwin

package source

import (
	"context"

	"github.com/shirou/gopsutil/v4/mem"
)

// memoryUsed 는 쓰는 메모리의 비율이다(전체에서 쓸 수 있는 메모리를 뺀 것).
func memoryUsed(ctx context.Context) (float64, bool) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, false
	}
	return v.UsedPercent / 100, true
}
