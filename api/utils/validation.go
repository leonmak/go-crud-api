package utils

import (
	"fmt"
	"github.com/google/uuid"
	"groupbuying.online/api/env"
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

func IsValidOrderByColumn(s string) bool {
	reqCols := []string{"posted_at", "total_price", "likes", "members"}
	for _, reqCol := range reqCols {
		if s == reqCol {
			return true
		}
	}
	return false
}

func IsValidOrderDirection(s string) bool {
	return s == "DESC" || s == "ASC"
}

func GetUserIdInSession(r *http.Request) (string, bool) {
	session, _ := env.Store.Get(r, env.Conf.SessionName)
	userId, ok := session.Values["userId"].(string)
	return userId, ok
}

func checkUserId(r *http.Request, w http.ResponseWriter, ownerId string)  {
	userId, ok := GetUserIdInSession(r)
	if ok && userId == ownerId {
		return
	}
	http.Error(w, "Forbidden", http.StatusForbidden)
	log.Fatalf("Not authenticated")
}