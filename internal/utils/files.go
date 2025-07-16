package utils

// IsHidden reports whether filename starts with a dot (hidden file on Unix systems).
func IsHidden(filename string) bool {
	return filename != "" && filename[0] == '.'
}
