package initialize

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"google.golang.org/appengine"
	"log"
	"os"

	"groupbuying.online/api/env"
	"groupbuying.online/api/routes"
	"groupbuying.online/api/structs"

	"firebase.google.com/go"
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
}

func initConfig() {
	var err error
	configFolder := "config/"
	if appengine.IsAppEngine() {
		configFolder = "../config/"
	}
	env.Conf, err = getConfiguration(configFolder, os.Getenv("ENV"))
	if err != nil {
		log.Fatal(err)
	}
	initFirebase(configFolder)
}

func initDB() {
	var err error
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		env.Conf.DBHost, env.Conf.DBPort, env.Conf.DBName, env.Conf.DBUsername, env.Conf.DBPassword)
	env.Db, err = sql.Open(env.Conf.DBDriver, connStr)
	env.Db.SetMaxIdleConns(100)
	if err != nil {
		log.Fatal(err)
	}
}

func initSessionStore() {
	key := []byte(env.Conf.SessionStoreKey)
	env.Store = sessions.NewCookieStore(key)
}

func getConfiguration(configFolder string, envType string) (*structs.Config, error) {
	if envType == "" {
		envType = "dev"
	}
	var configuration structs.Config
	file, err := os.Open(fmt.Sprintf("%s/%s.json", configFolder, envType))
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


func initFirebase(configFolder string) {
	opt := option.WithCredentialsFile(fmt.Sprintf("%s/serviceAccountKey.json", configFolder))
	var err error
	env.Firebase, err = firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
}