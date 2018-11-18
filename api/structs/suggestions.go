package structs

import "time"

type Suggestion struct {
	ID				string 		`json:"id",db:"id"`
	SearchString	*string 	`json:"searchString",db:"search_string"`
	PosterID        *string     `json:"posterId,omitEmpty",db:"poster_id"`
	CategoryID      *uint    	`json:"categoryId,omitempty",db:"category_id"`
	Latitude		*float64	`json:"latitude,omitempty",db:"latitude"`
	Longitude		*float64	`json:"longitude,omitempty",db:"longitude"`
	RadiusKm		*float64	`json:"radiusKm,omitempty",db:"radius_km"`
	BannerUrl 		*string 	`json:"bannerUrl,omitempty",db:"banner_url"`
	InactiveBy		*time.Time  `db:"inactive_by"`
}

