package utils

// IsHidden reports whether filename starts with a dot (hidden file on Unix systems).
func IsHidden(filename string) bool {
	return len(filename) > 0 && filename[0] == '.'
}
