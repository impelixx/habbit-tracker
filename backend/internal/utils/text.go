package utils

// TruncateText truncates text to maxLength and adds ellipsis if needed.
// If maxLength is less than 3, it returns the original text to avoid panic.
// For texts longer than maxLength, it truncates to (maxLength-3) and adds "...".
func TruncateText(text string, maxLength int) string {
	// Handle edge case where maxLength is too small
	if maxLength < 3 {
		return text
	}

	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}
