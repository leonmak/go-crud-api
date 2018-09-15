package config

type Configuration struct {
	Port 			int			`json:"port"`
	DBDriverName	string		`json:"dbDriverName"`
	DBSourceName	string		`json:"dbSourceName"`
	DBUsername		string		`json:"dbUsername"`
	DBPassword 		string		`json:"dbPassword"`
	SessionStoreKey string		`json:"sessionStoreKey"`
	SessionName	 	string		`json:"sessionName"`
	CSRFKey			string 		`json:"csrfKey"`

	FBAppId			string 		`json:"fbAppId"`
	FBAppSecret		string 		`json:"fbAppSecret"`
}
