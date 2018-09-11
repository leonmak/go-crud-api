package structs

// User & Deal has a city_id, consider sharding on cities' country / state
type City struct {
	ID 		uint	`json:"id",db:"id"`
	Name	string 	`json:"name",db:"name"`
	StateID	uint	`json:"stateId",db:"state_id"`
}

type State struct {
	ID 			uint	`json:"id",db:"id"`
	Name		string 	`json:"name",db:"name"`
	CountryID	uint	`json:"countryId",db:"country_id"`
}

type Country struct {
	ID 			uint	`json:"id",db:"id"`
	Name		string 	`json:"name",db:"name"`
	SortName	string	`json:"sortname",db:"sortname"`
}

