package eval

import (
	"encoding/binary"
	"fmt"
	"os"
)

// HR 是 Two Plus Two 手牌等级查找表。
// 在调用 Evaluate7 之前通过 Load() 加载以启用快速路径。
var HR []int32

// Load 从指定路径读取 HandRanks.dat 并填充 HR。
// 如需 T+2 性能，必须在任何评估之前调用一次。
// 如果未调用，Evaluate7 将回退到纯 Go 评估。
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("eval: open HandRanks.dat: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("eval: stat HandRanks.dat: %w", err)
	}
	n := info.Size() / 4
	HR = make([]int32, n)
	if err := binary.Read(f, binary.LittleEndian, HR); err != nil {
		return fmt.Errorf("eval: read HandRanks.dat: %w", err)
	}
	return nil
}

// Loaded 报告 T+2 查找表是否已加载。
func Loaded() bool { return len(HR) > 0 }
