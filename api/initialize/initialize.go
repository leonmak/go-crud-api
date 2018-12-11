package initialize

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"

	"groupbuying.online/api/env"
	"groupbuying.online/api/routes"
	"groupbuying.online/api/structs"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
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
	initFirebase()
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
	env.Db.SetMaxIdleConns(100)
	if err != nil {
		log.Fatal(err)
	}
}

func initSessionStore() {
	key := []byte(env.Conf.SessionStoreKey)
	env.Store = sessions.NewCookieStore(key)
}

func getConfiguration(envType string) (*structs.Config, error) {
	if envType == "" {
		envType = "dev"
	}
	var configuration structs.Config
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


func initFirebase() {
	opt := option.WithCredentialsFile("config/serviceAccountKey.json")
	var err error
	env.Firebase, err = firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
}