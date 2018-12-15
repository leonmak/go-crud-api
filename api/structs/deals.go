package structs

import (
	"time"
	"net/url"
)

// Maps to Deals table
type Deal struct {
	ID				string 		`json:"id",db:"id"`
	// uuid for dynamic tables for easier sharding
	Title			string		`json:"title",db:"title"`
	Description 	string		`json:"description",db:"description"`
	CategoryID      uint        `json:"categoryId",db:"category_id"`
	TotalPrice      *float32    `json:"totalPrice,omitempty",db:"total_price"`
	Quantity        *uint       `json:"quantity,omitempty",db:"quantity"`
	Benefits    	*string     `json:"benefits,omitempty",db:"benefits"`
	// pointer for possible nil values
	// first image in upload is thumbnailID
	ThumbnailUrl 	*string 	`json:"thumbnailUrl,omitempty",db:"thumbnail_id"`
	// location fields can be derived from lat lng (drop in) or text (reverse geocode) on POST
	Latitude		*float64	`json:"latitude,omitempty",db:"latitude"`
	Longitude		*float64	`json:"longitude,omitempty",db:"longitude"`
	// exact location text, open in maps
	LocationText    *string     `json:"locationText,omitempty",db:"location_text"`
	PosterID        string      `json:"posterId",db:"poster_id"`
	PostedAt        time.Time   `json:"postedAt",db:"posted_at"`
	UpdatedAt       *time.Time  `json:"updatedAt,omitempty",db:"updated_at"`
	InactiveAt      *time.Time  `json:"inactiveAt,omitempty",db:"inactive_at"`
	CountryCode		*string		`json:"countryCode",db:"country_code"`
	FeaturedUrl		*string		`json:"featuredUrl,omitEmpty",db:"featured_url"`
	// derived columns
	Likes			*uint		`json:"likes,omitEmpty"`
	Members 		*uint		`json:"members,omitEmpty"`
}

type DealCategory struct {
	ID				uint 	`json:"id",db:"id"`
	Name 			string 	`json:"name",db:"name"`
	IconUrl 		*string `json:"iconUrl,omitEmpty",db:"icon_url"`
	Priority 		uint 	`json:"priority",db:"priority"`
	DisplayName 	string 	`json:"displayName",db:"display_name"`
	IsActive	 	bool 	`json:"isActive",db:"is_active"`
}

type DealMembership struct {
	User		User		`json:"user"`
	DealID		string		`json:"dealId,omitempty",db:"deal_id"`
	JoinedAt	time.Time	`json:"joinedAt",db:"joined_at"`
}

type DealImage struct {
	ImageURL	url.URL		`json:"imageUrl",db:"image_url"`
	PosterID	string		`json:"posterId",db:"poster_id"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}

type DealLikes struct {
	ID			string 		`json:"id"`
	DealID		string		`json:"dealId,omitempty",db:"deal_id"`
	UserID		string		`json:"userId",db:"user_id"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
	IsUpVote	bool		`json:"isUpvote",db:"is_upvote"`
}

type DealComment struct {
	ID			string 		`json:"id",db:"id"`
	Username	string 		`json:"username",db:"username"`
	DealID		string		`json:"dealId,omitempty",db:"deal_id"`
	UserID		string 		`json:"userId",db:"user_id"`
	Comment		string		`json:"comment",db:"comment"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}
