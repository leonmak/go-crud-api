package main

import (
	"time"
	"net/url"
	"log"
	"net/http"
	"database/sql"
	"os"
	"encoding/json"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/lib/pq"

	"groupbuying.online/config"
)

// TODO: Implement:
// TODO: - csrf, rate limit middleware
// TODO: - cloudinary, email verification, change name/password
type User struct {
	ID				string 		`json:"id,omitempty",db:"id"`
	DisplayName 	string		`json:"displayName,omitEmpty",db:"display_name"`
	ImageURL		string 		`json:"imageUrl,omitEmpty",db:"image_url"`
	Email	 		string		`json:"email,omitEmpty",db:"email"`
	PasswordDigest 	string 		`json:"passwordDigest,omitEmpty",db:"password_digest"`
	VerifyAttempt	time.Time	`json:"verifyAttempt,omitEmpty",db:"verify_attempt"`
	Verified		bool		`json:"verified,omitEmpty",db:"verified"`
}

// For marshalling login / register requests
type UserCredentials struct {
	Email 		string	`json:"email"`
	Password	string	`json:"password"`
	DisplayName string	`json:"displayName,omitEmpty"`
}

type Deal struct {
	ID			string 		`json:"id",db:"id"`
	Title		string		`json:"title",db:"title"`
	Description string		`json:"description",db:"description"`
	Lat			float64		`json:"latitude",db:"latitude"`
	Long		float64		`json:"longitude",db:"longitude"`
	PosterID	string		`json:"posterId",db:"poster_id"`
	CreatedAt	time.Time	`json:"createdAt",db:"created_at"`
	UpdatedAt	time.Time	`json:"updatedAt",db:"updated_at"`
	ExpireBy	time.Time	`json:"expireBy",db:"expire_by"`
	Expired		bool		`json:"expired",db:"expired"`
}

type DealMembership struct {
	ID			string 		`json:"id",db:"id"`
	UserID		string		`json:"userId",db:"user_id"`
	DealID		string		`json:"dealId",db:"deal_id"`
	JoinedAt	time.Time	`json:"joinedAt",db:"joined_at"`
}

type DealImage struct {
	ID			string 		`json:"id",db:"id"`
	ImageUrl	url.URL		`json:"imageUrl",db:"image_url"`
	PosterId	string		`json:"posterId",db:"poster_id"`
}

type DealVote struct {
	ID		string 		`json:"id"`
	IsUp 	bool		`json:"isUp",db:"is_up"`
	UserID	string		`json:"userId",db:"user_id"`
}

type DealComment struct {
	ID		string 	`json:"id",db:"id"`
	UserID	string 	`json:"userId",db:"user_id"`
	Text	string	`json:"text",db:"text"`
}

var conf *config.Configuration
var db *sql.DB
var store *sessions.CookieStore


func main() {
	initConfig()
	initDB()
	initSessionStore()
	initRouter()
}

func initRouter() {
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/deals", GetDeals).Methods("GET")
	api.HandleFunc("/deal/{id}", use(GetDeal, auth)).Methods("GET")
	api.HandleFunc("/deal/{id}", use(PostDeal, auth)).Methods("POST")
	api.HandleFunc("/deal/{id}", use(DeleteDeal, auth)).Methods("DELETE")

	api.HandleFunc("/register", CreateUser).Methods("POST")
	api.HandleFunc("/login", LoginUser).Methods("POST")
	api.HandleFunc("/logout", use(LogoutUser, auth)).Methods("POST")

	fmt.Printf("listening on %d\n", conf.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), api))
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

// Decorate the request handler
func use(h http.HandlerFunc, middleware ...Middleware) http.HandlerFunc {
	//  r.HandleFunc("/login", use(LoginUser, rateLimit, csrf))
	for _, m := range middleware {
		h = m(h)
	}
	return h
}

func auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, conf.SessionName)
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

func GetDeals(w http.ResponseWriter, r *http.Request) {

}

func GetDeal(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "The cake is a lie!")
}

func PostDeal(w http.ResponseWriter, r *http.Request) {

}

func DeleteDeal(w http.ResponseWriter, r *http.Request) {

}

func LogoutUser(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, conf.SessionName)
	session.Values["authenticated"] = false
	session.Save(r, w)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	creds := &UserCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil || creds.DisplayName == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Invalid submission.")
		return
	}
	passwordDigest, err := HashPassword(creds.Password)
	_, err = db.Query("insert into users (email, password_digest, display_name) values ($1, $2, $3)",
		creds.Email, passwordDigest, creds.DisplayName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Server Error.")
		return
	}
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, conf.SessionName)
	creds := &UserCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("Attempted login with email: %s", creds.Email)

	var passwordDigest string
	err = db.QueryRow("select password_digest from users where email=$1",
		creds.Email).Scan(&passwordDigest)
	switch err {
	case sql.ErrNoRows:
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Login Failed as user doesn't exist")
	case nil:
		if !CheckPasswordHash(creds.Password, passwordDigest) {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("Login Failed as password mismatched")
		}
		// Save authenticated session if successful
		session.Values["authenticated"] = true
		session.Save(r, w)
		log.Printf("Login Successful")
	default:
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Login Failed as db errored. %v", err)
	}
}

func initSessionStore() {
	key := []byte(conf.SessionStoreKey)
	store = sessions.NewCookieStore(key)
}

func initDB() {
	var err error
	connStr := fmt.Sprintf("dbname=%s user=%s password=%s sslmode=disable",
		conf.DBSourceName, conf.DBUsername, conf.DBPassword)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
}

func getConfiguration(env string) (*config.Configuration, error) {
	if env == "" {
		env = "dev"
	}
	var configuration config.Configuration
	file, err := os.Open("config/" + env + ".json")
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

func initConfig() {
	var err error
	conf, err = getConfiguration(os.Getenv("GO_ENV"))
	if err != nil {
		log.Fatal(err)
	}
}
