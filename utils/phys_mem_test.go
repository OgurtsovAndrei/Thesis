package utils

import (
	"testing"
	"time"
)

func TestMemoryMonitor(t *testing.T) {
	ForceGC()
	initial, _ := PhysicalMemStats()
	t.Logf("Initial RSS: %d KB", initial/1024)

	m := StartMemoryMonitor(1 * time.Millisecond)
	
	// Allocate some memory
	data := make([]byte, 100*1024*1024) // 100 MB
	for i := range data {
		data[i] = byte(i)
	}
	
	time.Sleep(10 * time.Millisecond)
	peak := m.Stop()
	t.Logf("Peak RSS during 100MB alloc: %d KB", peak/1024)
	
	if peak <= initial {
		t.Errorf("Peak RSS (%d) should be greater than initial (%d)", peak, initial)
	}
	
	// Ensure data is not GC'd too early
	_ = data[0]
}
