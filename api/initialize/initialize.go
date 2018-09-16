package initialize

import (
	"log"
	"database/sql"
	"os"
	"encoding/json"
	"fmt"

	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"

	"groupbuying.online/config"
	"groupbuying.online/api/routes"
	"groupbuying.online/api/env"
)

// TODO: Implement:
// TODO: - csrf, rate limit middleware
// TODO: - cloudinary, email verification, change name/password

func Init() {
	initEnv()
	routes.InitRouter()
}

func initEnv() {
	initConfig()
	initDB()
	initSessionStore()
}

func initConfig() {
	var err error
	env.Conf, err = getConfiguration(os.Getenv("GO_ENV"))
	if err != nil {
		log.Fatal(err)
	}
}

func initDB() {
	var err error
	connStr := fmt.Sprintf("dbname=%s user=%s password=%s sslmode=disable",
		env.Conf.DBSourceName, env.Conf.DBUsername, env.Conf.DBPassword)
	env.Db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
}

func initSessionStore() {
	key := []byte(env.Conf.SessionStoreKey)
	env.Store = sessions.NewCookieStore(key)
}

func getConfiguration(envType string) (*config.Configuration, error) {
	if envType == "" {
		envType = "dev"
	}
	var configuration config.Configuration
	file, err := os.Open("config/" + envType + ".json")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	if err != nil {
		return nil, err
	}
	return &configuration, err
}
