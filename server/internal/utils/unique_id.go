package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// GenerateUniqueID generates a unique ID in format #WORD-123
func GenerateUniqueID(name string) string {
	// Take first word of name and capitalize
	words := strings.Fields(name)
	prefix := strings.ToUpper(words[0])
	if len(prefix) > 6 {
		prefix = prefix[:6]
	}

	// Generate random 3-digit number
	rand.Seed(time.Now().UnixNano())
	number := rand.Intn(900) + 100 // 100-999

	return fmt.Sprintf("#%s-%d", prefix, number)
}

// ValidateUniqueID validates the format of a unique ID
func ValidateUniqueID(uniqueID string) bool {
	// Should start with # and contain a dash
	if len(uniqueID) < 5 || uniqueID[0] != '#' {
		return false
	}

	parts := strings.Split(uniqueID[1:], "-")
	return len(parts) == 2
}
