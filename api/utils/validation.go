package utils

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"unicode"
)

func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func WriteError(w http.ResponseWriter, message string) {
	log.Println(message)
	WriteJsonResponse(w, "error", message)
}

func IsValidUsername(s string) error {
	maxLength := 33
	if len(s) > maxLength {
		return fmt.Errorf("username more than %d characters", maxLength)
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return fmt.Errorf("username not alphanumeric")
		}
	}
	return nil
}