package utils

import (
	"fmt"
	"os"
	"sync"
)

var (
	statsFile = "candidate_stats.log"
	statsMu   sync.Mutex
)

func LogCandidateMatch(testName string, matchCounts []int) {
	statsMu.Lock()
	defer statsMu.Unlock()

	f, err := os.OpenFile(statsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	line := fmt.Sprintf("%s", testName)
	for _, count := range matchCounts {
		line += fmt.Sprintf(",%d", count)
	}
	fmt.Fprintln(f, line)
}

func ClearStats() {
	statsMu.Lock()
	defer statsMu.Unlock()
	os.Remove(statsFile)
}
