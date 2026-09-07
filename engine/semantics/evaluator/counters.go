package evaluator

import "sync"

// Counters is the coverage state used by the test command. Profiling accounting
// no longer lives here; the profiling network is the sole Nezha accounting
// substrate after Cut P3D.
type Counters struct {
	mu       sync.Mutex
	coverage map[string]int64
}

// NewCoverageCounters creates a counter block with coverage tracking enabled.
func NewCoverageCounters() *Counters {
	return &Counters{coverage: make(map[string]int64)}
}

// CoverHit records a coverage hit at the given file and line.
func (c *Counters) CoverHit(file string, line int) {
	if c == nil || c.coverage == nil {
		return
	}
	c.mu.Lock()
	c.coverage[coverKey(file, line)]++
	c.mu.Unlock()
}

// Coverage returns a snapshot of the coverage map.
func (c *Counters) Coverage() map[string]int64 {
	if c == nil || c.coverage == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int64, len(c.coverage))
	for k, v := range c.coverage {
		out[k] = v
	}
	return out
}

func coverKey(file string, line int) string {
	buf := make([]byte, 0, len(file)+12)
	buf = append(buf, file...)
	buf = append(buf, ':')
	buf = appendInt(buf, line)
	return string(buf)
}

func appendInt(buf []byte, n int) []byte {
	if n == 0 {
		return append(buf, '0')
	}
	if n < 0 {
		buf = append(buf, '-')
		n = -n
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return append(buf, digits[i:]...)
}
