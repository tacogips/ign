package parser

// processRawDirective returns the raw content without processing.
// This allows literal @ign-* syntax to appear in output.
func processRawDirective(args string) (string, error) {
	// Raw directive simply returns its content as-is
	// No variable substitution or directive processing
	return args, nil
}
