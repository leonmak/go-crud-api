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
	// pointer for possible nil values
	// first image in upload is thumbnailID
	ThumbnailID		*string 	`json:"thumbnailId,omitempty",db:"thumbnail_id"`
	// location fields can be derived from lat lng (drop in) or text (reverse geocode) on POST
	Latitude		*float64	`json:"latitude,omitempty",db:"latitude"`
	Longitude		*float64	`json:"longitude,omitempty",db:"longitude"`
	// exact location text, open in maps
	LocationText	*string 	`json:"locationText,omitempty",db:"location_text"`
	TotalPrice		*float32	`json:"totalPrice,omitempty",db:"total_price"`
	TotalSavings	*float32	`json:"totalSavings,omitempty",db:"total_savings"`
	Quantity		*uint		`json:"quantity,omitempty",db:"quantity"`
	CategoryID		uint		`json:"categoryId",db:"category_id"`
	PosterID		string		`json:"posterId",db:"poster_id"`
	PostedAt		time.Time	`json:"postedAt",db:"posted_at"`
	UpdatedAt		*time.Time	`json:"updatedAt,omitempty",db:"updated_at"`
	InactiveAt		*time.Time	`json:"inactiveAt,omitempty",db:"inactive_at"`
	CityID			uint		`json:"cityId",db:"city_id"`
}

type DealCategory struct {
	ID				uint 	`json:"id",db:"id"`
	Name 			string 	`json:"name",db:"name"`
	MaxImages		uint	`json:"maxImages",db:"max_images"`
	MaxActiveDays	uint	`json:"maxActiveDays",db:"max_active_days"`
}

type DealMembership struct {
	User		User		`json:"user"`
	DealID		string		`json:"dealId",db:"deal_id"`
	JoinedAt	time.Time	`json:"joinedAt",db:"joined_at"`
	LeftAt		*time.Time	`json:"leftAt,omitEmpty",db:"left_at"`
}

type DealImage struct {
	ImageURL	url.URL		`json:"imageUrl",db:"image_url"`
	PosterID	string		`json:"posterId",db:"poster_id"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}

type DealLikes struct {
	ID			string 		`json:"id"`
	DealID		string		`json:"dealId",db:"deal_id"`
	UserID		string		`json:"userId",db:"user_id"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
	IsUpVote	bool		`json:"isUpvote",db:"is_upvote"`
}

type DealComment struct {
	Username	string 		`json:"username"`
	DealID		string		`json:"dealId",db:"deal_id"`
	UserID		string 		`json:"userId",db:"user_id"`
	Comment		string		`json:"comment",db:"comment"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}

