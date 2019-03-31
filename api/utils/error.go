package utils

import (
	"log"
	"net/http"
)

// Throws an error after logging and writing to response
func CheckFatalError(w http.ResponseWriter, err error) {
	if err != nil {
		WriteErrorJsonResponse(w, err.Error())
		log.Fatal(err.Error())
	}
}
