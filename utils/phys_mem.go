package utils

import (
	"runtime"
	"time"
)

// PhysicalMemStats returns the current and peak Resident Set Size (RSS) of the process.
// Note: current implementation for Darwin/Linux.
func PhysicalMemStats() (currentRSS, peakRSS uint64) {
	return getPhysicalMemStats()
}

// MemoryMonitor tracks peak RSS during a task.
type MemoryMonitor struct {
	done chan struct{}
	peak uint64
}

// StartMemoryMonitor begins polling RSS in the background.
func StartMemoryMonitor(interval time.Duration) *MemoryMonitor {
	m := &MemoryMonitor{
		done: make(chan struct{}),
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				current, _ := PhysicalMemStats()
				if current > m.peak {
					m.peak = current
				}
			case <-m.done:
				return
			}
		}
	}()
	return m
}

// Stop returns the peak RSS observed since StartMemoryMonitor.
func (m *MemoryMonitor) Stop() uint64 {
	close(m.done)
	// One last sample to be sure
	current, _ := PhysicalMemStats()
	if current > m.peak {
		m.peak = current
	}
	return m.peak
}

// SystemResidentMemory returns physical memory used by the process in bytes.
// It is more accurate than MemStats for CGo applications.
func SystemResidentMemory() uint64 {
	rss, _ := PhysicalMemStats()
	return rss
}

// ForceGC runs a heavy garbage collection to stabilize RSS.
func ForceGC() {
	runtime.GC()
	runtime.Gosched()
	// Give OS a moment to reclaim pages
	time.Sleep(10 * time.Millisecond)
}
