/*
Copyright 2025 The OADP CLI Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shared

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Indent adds a prefix to each non-empty line of text
func Indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if len(line) > 0 {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// ColorizePhase returns the phase string with ANSI color codes
func ColorizePhase(phase string) string {
	const (
		colorGreen  = "\033[32m"
		colorYellow = "\033[33m"
		colorRed    = "\033[31m"
		colorReset  = "\033[0m"
	)

	switch phase {
	case "Completed":
		return colorGreen + phase + colorReset
	case "InProgress", "New":
		return colorYellow + phase + colorReset
	case "Failed", "FailedValidation", "PartiallyFailed":
		return colorRed + phase + colorReset
	default:
		return phase
	}
}

// PrintLabelsOrAnnotations prints labels or annotations in oc-style format
// with proper alignment for multi-line output
func PrintLabelsOrAnnotations(out io.Writer, fieldName string, items map[string]string) {
	fmt.Fprintf(out, "%s", fieldName)
	if len(items) == 0 {
		fmt.Fprintf(out, "<none>\n")
	} else {
		keys := make([]string, 0, len(items))
		for k := range items {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Calculate padding for alignment (field name length)
		padding := strings.Repeat(" ", len(fieldName))

		for i, k := range keys {
			if i == 0 {
				fmt.Fprintf(out, "%s=%s\n", k, items[k])
			} else {
				fmt.Fprintf(out, "%s%s=%s\n", padding, k, items[k])
			}
		}
	}
}
