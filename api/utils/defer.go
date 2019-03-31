package utils

import (
	"database/sql"
	"log"
	"net/http"
)

func CloseResponse(resp *http.Response) {
	if err := resp.Body.Close(); err != nil {
		log.Fatal("failed to close connection")
	}
}

func CloseRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		log.Fatal("failed to close rows")
	}
}
