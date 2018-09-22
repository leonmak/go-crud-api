package structs

// Maps to Users table
type User struct {
	ID				string 		`json:"id",db:"id"`
	DisplayName 	string		`json:"displayName",db:"display_name"`
	ImageURL		string 		`json:"imageUrl",db:"image_url"`
}

// Temp struct For marshalling login / register requests
type UserCredentials struct {
	Email 		string	`json:"email"`
	Password	string	`json:"password"`
	DisplayName string	`json:"displayName"`
}

type UserCredentialSocialMedia struct {
	Email 		string	`json:"email"`
	DisplayName string	`json:"displayName"`
	ImageUrl 	string  `json:"imageUrl"`
}

type SocialSignInCredentials struct {
	UserID	 	string 	`json:"userId"`
	UserToken 	string 	`json:"userToken"`
	Email 		string	`json:"email"`
}
