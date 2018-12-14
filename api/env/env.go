package env

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
	"groupbuying.online/api/structs"
	"log"
	"os"

	_ "github.com/lib/pq"

	"firebase.google.com/go"
	"google.golang.org/api/option"
)

var (
	Conf *structs.Config
	Db *sql.DB
	Store *sessions.CookieStore
	Firebase *firebase.App
)


func InitEnv() {
	initConfig()
	initDB()
	initSessionStore()
}

func initConfig() {
	var err error
	configFolder := "config/"
	Conf, err = getConfiguration(configFolder, os.Getenv("ENV"))
	if err != nil {
		log.Fatal(err)
	}
	initFirebase(configFolder)
}

func initDB() {
	var err error
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		Conf.DBHost, Conf.DBPort, Conf.DBName, Conf.DBUsername, Conf.DBPassword)
	Db, err = sql.Open(Conf.DBDriver, connStr)
	Db.SetMaxIdleConns(100)
	if err != nil {
		log.Fatal(err)
	}
}

func initSessionStore() {
	key := []byte(Conf.SessionStoreKey)
	Store = sessions.NewCookieStore(key)
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
	Firebase, err = firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
}