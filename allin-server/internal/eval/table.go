package eval

import (
	"encoding/binary"
	"fmt"
	"os"
)

// HR is the Two Plus Two hand rank lookup table.
// Load via Load() before calling Evaluate7 for the fast path.
var HR []int32

// Load reads HandRanks.dat from path and populates HR.
// Must be called once before any evaluation if T+2 performance is desired.
// If not called, Evaluate7 falls back to pure Go evaluation.
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

// Loaded reports whether the T+2 lookup table has been loaded.
func Loaded() bool { return len(HR) > 0 }
