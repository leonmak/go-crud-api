package routes

import (
	"database/sql"
	"encoding/json"
	"groupbuying.online/api/env"
	"groupbuying.online/api/structs"
	"groupbuying.online/api/utils"
	"net/http"
	"time"
)

func getSuggestions(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	after := values.Get("after")
	iso8601Layout := "2006-01-02T15:04:05Z"
	afterT, err := time.Parse(iso8601Layout, after)
	var suggestions []structs.Suggestion
	var rows *sql.Rows

	rows, err = env.Db.Query("SELECT search_string, poster_id, category_id, latitude, longitude, radius_km, banner_url FROM suggestions WHERE active_from < $1 AND $1 < inactive_by", afterT)
	defer rows.Close()
	for rows.Next() {
		var s structs.Suggestion
		err = rows.Scan(&s.SearchString, &s.PosterID, &s.CategoryID, &s.Latitude, &s.Longitude, &s.RadiusKm, &s.BannerUrl)
		if err != nil {
			utils.WriteErrorJsonResponse(w, err.Error())
			return
		}
		suggestions = append(suggestions, s)
	}
	suggestionsArr, err := json.Marshal(suggestions)
	if string(suggestionsArr) == "null" {
		suggestionsArr = []byte("[]")
	}
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
	} else {
		w.Write(suggestionsArr)
	}
}