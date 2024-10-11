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