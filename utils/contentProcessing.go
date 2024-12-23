package utils
import (
	"strings"
)
func CleanString(s string) string {
	// Use Fields to split the string into slices of words, automatically removes extra spaces
	words := strings.Fields(s)
	// Join the words back together with a single space
	return strings.Join(words, " ")
}

func CountToken(s string)int{
	// Clean the string first to normalize spaces
	cleanedString := CleanString(s)
	// Split the string into words based on spaces
	words := strings.Fields(cleanedString)
	// Return the number of words
	return len(words)
}
func TrimString(s string) string {
	// Check the length of the string
	if len(s) < 25 {
		// If the string is shorter than 15 characters, return it as is
		return s
	}
	// If the string is 15 or more characters, return only the first 15 characters
	return s[:25]
}