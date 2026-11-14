package cli

import (
	"fmt"
	"os"
)

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[90m"
)

// Output formatting helpers

// printInfo prints an informational message
func printInfo(msg string) {
	if globalQuiet {
		return
	}
	fmt.Println(msg)
}

// printSuccess prints a success message
func printSuccess(msg string) {
	if globalQuiet {
		return
	}
	if globalNoColor {
		fmt.Printf("✓ %s\n", msg)
	} else {
		fmt.Printf("%s✓%s %s\n", colorGreen, colorReset, msg)
	}
}

// printWarning prints a warning message
func printWarning(msg string) {
	if globalQuiet {
		return
	}
	if globalNoColor {
		fmt.Printf("⚠ %s\n", msg)
	} else {
		fmt.Printf("%s⚠%s %s\n", colorYellow, colorReset, msg)
	}
}

// printErrorMsg prints an error message (different from printError which takes error type)
func printErrorMsg(msg string) {
	if globalNoColor {
		fmt.Fprintf(os.Stderr, "✗ %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "%s✗%s %s\n", colorRed, colorReset, msg)
	}
}

// printVerbose prints a verbose message (only if verbose is enabled)
func printVerbose(verbose bool, msg string) {
	if !verbose || globalQuiet {
		return
	}
	if globalNoColor {
		fmt.Printf("[VERBOSE] %s\n", msg)
	} else {
		fmt.Printf("%s[VERBOSE]%s %s\n", colorGray, colorReset, msg)
	}
}

// printDebug prints a debug message
func printDebug(msg string) {
	if globalQuiet {
		return
	}
	if globalNoColor {
		fmt.Printf("[DEBUG] %s\n", msg)
	} else {
		fmt.Printf("%s[DEBUG]%s %s\n", colorCyan, colorReset, msg)
	}
}

// printProgress prints a progress indicator
func printProgress(msg string) {
	if globalQuiet {
		return
	}
	if globalNoColor {
		fmt.Printf("→ %s\n", msg)
	} else {
		fmt.Printf("%s→%s %s\n", colorBlue, colorReset, msg)
	}
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// printHeader prints a section header
func printHeader(title string) {
	if globalQuiet {
		return
	}
	if globalNoColor {
		fmt.Printf("\n=== %s ===\n", title)
	} else {
		fmt.Printf("\n%s=== %s ===%s\n", colorMagenta, title, colorReset)
	}
}

// printSeparator prints a separator line
func printSeparator() {
	if globalQuiet {
		return
	}
	if globalNoColor {
		fmt.Println("────────────────────────────────────────")
	} else {
		fmt.Printf("%s────────────────────────────────────────%s\n", colorGray, colorReset)
	}
}
