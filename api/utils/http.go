package utils

import (
	"log"
	"net/http"
)

func CloseResponse(resp *http.Response) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal("failed to close connection")
		}
	}()
}


