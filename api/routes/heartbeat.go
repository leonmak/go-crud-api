package routes

import (
	"groupbuying.online/api/utils"
	"net/http"
)

func heartbeat(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	utils.WriteString(w, "{}")
}