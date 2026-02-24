package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MemReport provides a detailed, hierarchical memory usage report for a component.
type MemReport struct {
	Name       string      `json:"name"`
	TotalBytes int         `json:"total_bytes"`
	Children   []MemReport `json:"children,omitempty"`
}

// Print formats and prints the MemReport as a tree.
func (r MemReport) Print(indent int) {
	prefix := strings.Repeat("  ", indent)
	fmt.Printf("%s- %s: %d bytes\n", prefix, r.Name, r.TotalBytes)
	for _, child := range r.Children {
		child.Print(indent + 1)
	}
}

// JSON returns a JSON string representation of the MemReport.
func (r MemReport) JSON() string {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}
	return string(b)
}

// String returns a string representation of the MemReport as a tree.
func (r MemReport) String() string {
	var sb strings.Builder
	r.buildString(&sb, 0)
	return sb.String()
}

func (r MemReport) buildString(sb *strings.Builder, indent int) {
	prefix := strings.Repeat("  ", indent)
	sb.WriteString(fmt.Sprintf("%s- %s: %d bytes\n", prefix, r.Name, r.TotalBytes))
	for _, child := range r.Children {
		child.buildString(sb, indent+1)
	}
}
