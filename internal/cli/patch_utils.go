package cli

import (
	"bytes"
	"fmt"
	"strings"
)

// countLines counts the number of lines in content.
func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := bytes.Count(content, []byte{'\n'})
	// If content doesn't end with newline, there's one more line
	if len(content) > 0 && content[len(content)-1] != '\n' {
		count++
	}
	return count
}

// printPatchContent prints file content with + prefix for each line.
func printPatchContent(content []byte) {
	if len(content) == 0 {
		return
	}

	// Check if content is likely binary
	if isBinaryContent(content) {
		fmt.Println("+[binary file]")
		return
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		// Skip the last empty line if content ends with newline
		if i == len(lines)-1 && line == "" {
			continue
		}
		fmt.Printf("+%s\n", line)
	}
}

// isBinaryContent checks if content appears to be binary.
func isBinaryContent(content []byte) bool {
	// Check first 512 bytes for null bytes
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}
