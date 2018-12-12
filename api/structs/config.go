package structs

type Config struct {
	Port 			int			`json:"port"`
	DBHost			string		`json:"dbHost"`
	DBPort			int			`json:"dbPort"`
	DBName			string		`json:"dbName"`
	DBDriver		string		`json:"dbDriver"`
	DBUsername		string		`json:"dbUsername"`
	DBPassword 		string		`json:"dbPassword"`
	SessionStoreKey string		`json:"sessionStoreKey"`
	SessionName	 	string		`json:"sessionName"`
	CSRFKey			string 		`json:"csrfKey"`

	FBAppId			string 		`json:"fbAppId"`
	FBAppSecret		string 		`json:"fbAppSecret"`
}


type CloudSQLConfig struct {
	Username, Password, Instance string
}