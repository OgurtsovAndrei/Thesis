package perf

import (
	"runtime"
	"fmt"
)

// Stats holds hardware performance counter values.
type Stats struct {
	Cycles       uint64
	Instructions uint64
	L1Misses     uint64
	L2Misses     uint64 // Often mapped to LLC/L3 on some systems
}

// Monitor handles starting and stopping hardware counters.
type Monitor interface {
	Stop() Stats
}

// Start initiates hardware counter collection for the current thread.
// On macOS, it requires root privileges and uses private kperf API.
// On Linux, it uses perf_event_open.
func Start() Monitor {
	runtime.LockOSThread() // Counters are thread-local
	return startMonitor()
}

func (s Stats) String() string {
	ipc := 0.0
	if s.Cycles > 0 {
		ipc = float64(s.Instructions) / float64(s.Cycles)
	}
	return fmt.Sprintf("Cycles: %d, Instr: %d, IPC: %.2f, L1-Miss: %d, L2-Miss: %d",
		s.Cycles, s.Instructions, ipc, s.L1Misses, s.L2Misses)
}
