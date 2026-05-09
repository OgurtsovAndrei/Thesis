//go:build linux
package utils

import (
	"fmt"
	"os"
)

func getPhysicalMemStats() (currentRSS, peakRSS uint64) {
	var rssPages uint64
	f, err := os.Open("/proc/self/statm")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	// statm format: size resident shared text lib data dt
	fmt.Fscanf(f, "%d %d", &rssPages, &rssPages)
	
	pageSize := uint64(os.Getpagesize())
	return rssPages * pageSize, 0
}
