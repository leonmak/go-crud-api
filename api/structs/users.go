package structs

// Maps to Users table
type User struct {
	ID				string 		`json:"id",db:"id"`
	DisplayName 	string		`json:"displayName",db:"display_name"`
	ImageURL		*string 	`json:"imageUrl,omitempty",db:"image_url"`
	CountryCode		string 		`json:"countryCode",db:"country_code"`
	Reputation		int	 		`json:"reputation",db:"reputation"`
}

// Temp struct For marshalling login / register requests
type UserCredentials struct {
	Email 		string	`json:"email"`
	Token		string 	`json:"token"`
	DisplayName string	`json:"displayName"`
	CountryCode	string 	`json:"countryCode"`
}

type UserCredentialSocialMedia struct {
	Email 		string	`json:"email"`
	DisplayName string	`json:"displayName"`
	ImageUrl 	string  `json:"imageUrl"`
	AuthType	string  `json:"authType"`
	CountryCode	string 	`json:"countryCode"`
}

type SocialSignInCredentials struct {
	UserID	 	string 	`json:"userId"`
	UserToken 	string 	`json:"userToken"`
	Email 		string	`json:"email"`
}
