package app

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateGitRef validates a git branch, tag, or commit reference.
func ValidateGitRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("git reference cannot be empty")
	}
	if strings.TrimSpace(ref) != ref {
		return fmt.Errorf("invalid git reference %q: leading or trailing whitespace is not allowed", ref)
	}
	if strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/") {
		return fmt.Errorf("invalid git reference %q: cannot start or end with '/'", ref)
	}
	if strings.HasSuffix(ref, ".") {
		return fmt.Errorf("invalid git reference %q: cannot end with '.'", ref)
	}
	if strings.Contains(ref, "..") {
		return fmt.Errorf("invalid git reference %q: cannot contain '..'", ref)
	}
	if strings.Contains(ref, "//") {
		return fmt.Errorf("invalid git reference %q: cannot contain '//'", ref)
	}
	if strings.Contains(ref, "@{") {
		return fmt.Errorf("invalid git reference %q: cannot contain '@{'", ref)
	}
	if strings.HasSuffix(ref, ".lock") || strings.Contains(ref, ".lock/") {
		return fmt.Errorf("invalid git reference %q: cannot contain a .lock path component", ref)
	}

	for _, r := range ref {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("invalid git reference %q: whitespace and control characters are not allowed", ref)
		}
		switch r {
		case '~', '^', ':', '?', '*', '[', '\\':
			return fmt.Errorf("invalid git reference %q: contains invalid character %q", ref, r)
		}
	}

	for _, part := range strings.Split(ref, "/") {
		if part == "" {
			return fmt.Errorf("invalid git reference %q: empty path component", ref)
		}
		if strings.HasPrefix(part, ".") {
			return fmt.Errorf("invalid git reference %q: path components cannot start with '.'", ref)
		}
	}

	return nil
}
