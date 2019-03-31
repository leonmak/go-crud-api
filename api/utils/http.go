package utils

import (
	"encoding/json"
	"net/http"
)

func WriteBytes(w http.ResponseWriter, message []byte) {
	_, err := w.Write(message)
	CheckFatalError(w, err)
}

func WriteString(w http.ResponseWriter, message string) {
	WriteBytes(w, []byte(message))
}

func WriteStructs(w http.ResponseWriter, instance interface{}) {
	instanceBytes, err := json.Marshal(instance)
	CheckFatalError(w, err)
	WriteBytes(w, instanceBytes)
}