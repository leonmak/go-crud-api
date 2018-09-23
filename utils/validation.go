package utils

import (
	"github.com/google/uuid"
	"net/http"
	"fmt"
	"unicode"
)

func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func WriteError(w http.ResponseWriter, message string) {
	w.Write([]byte(fmt.Sprintf(`{"error": "%s"`, message)))
}

func IsValidUsername(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}