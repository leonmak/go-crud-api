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
	"github.com/teris-io/shortid"
	"strconv"
	"strings"
)

// TODO: Implement:
// TODO: - csrf, rate limit middleware
// TODO: - cloudinary, email verification, change name/password

// Maps to Users table
type User struct {
	// uuid for dynamic tables for easier sharding
	ID					string 		`json:"id,omitempty",db:"id"`
	Email	 			string		`json:"email",db:"email"`
	DisplayName 		string		`json:"displayName",db:"display_name"`
	URLAlias 			string 		`json:"urlAlias",db:"url_alias"`
	PasswordDigest		string 		`json:"passwordDigest",db:"password_digest"`
	ImageURL			string 		`json:"imageUrl",db:"image_url"`
	VerifyEmailSentAt	time.Time	`json:"verifyEmailSentAt",db:"verify_email_sent_at"`
	VerifiedAt			bool		`json:"verifiedAt",db:"verified_at"`
	CityID				uint16		`json:"cityId",db:"city_id"`
}

// Temp struct For marshalling login / register requests
type UserCredentials struct {
	Email 		string	`json:"email"`
	Password	string	`json:"password"`
	DisplayName string	`json:"displayName"`
}

// Maps to Deals table
type Deal struct {
	// uuid for dynamic tables for easier sharding
	Title			string		`json:"title",db:"title"`
	Description 	string		`json:"description",db:"description"`
	URLAlias 		string 		`json:"urlAlias",db:"url_alias"`
	// location fields can be derived from lat lng (drop in) or text (reverse geocode) on POST
	Latitude		*float64	`json:"latitude,omitempty",db:"latitude"`
	Longitude		*float64	`json:"longitude,omitempty",db:"longitude"`
	// exact location text, open in maps
	LocationText	*string 	`json:"locationText,omitempty",db:"location_text"`
	ExpectedPrice	*float32	`json:"expectedPrice,omitempty",db:"expected_price"`
	CategoryID		uint16		`json:"categoryId",db:"category_id"`
	PosterID		string		`json:"posterId",db:"poster_id"`
	PostedAt		time.Time	`json:"postedAt",db:"posted_at"`
	UpdatedAt		*time.Time	`json:"updatedAt,omitempty",db:"updated_at"`
	InactiveAt		*time.Time	`json:"inactiveAt,omitempty",db:"inactive_at"`
	CityID			uint16		`json:"cityId",db:"city_id"`
}

type DealCategory struct {
	ID				uint16 	`json:"id",db:"id"`
	Name 			string 	`json:"name",db:"name"`
	MaxImages		uint8	`json:"maxImages",db:"max_images"`
	MaxActiveDays	uint8	`json:"maxActiveDays",db:"max_active_days"`
}

type DealMembership struct {
	UserID		string		`json:"userId",db:"user_id"`
	DealID		string		`json:"dealId",db:"deal_id"`
	JoinedAt	time.Time	`json:"joinedAt",db:"joined_at"`
	LeftAt		time.Time	`json:"leftAt",db:"left_at"`
}

type DealImage struct {
	ID			string 		`json:"id",db:"id"`
	DealID		string		`json:"dealId",db:"deal_id"`
	ImageURL	url.URL		`json:"imageUrl",db:"image_url"`
	PosterID	string		`json:"posterId",db:"poster_id"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}

type DealVote struct {
	ID			string 		`json:"id"`
	DealID		string		`json:"dealId",db:"deal_id"`
	UserID		string		`json:"userId",db:"user_id"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}

type DealComment struct {
	ID			string 		`json:"id",db:"id"`
	DealID		string		`json:"dealId",db:"deal_id"`
	UserID		string 		`json:"userId",db:"user_id"`
	Comment		string		`json:"comment",db:"comment"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}

// User & Deal has a city_id, consider sharding on cities' country / state
type City struct {
	ID 		uint16	`json:"id",db:"id"`
	Name	string 	`json:"name",db:"name"`
	StateID	uint16	`json:"stateId",db:"state_id"`
}

type State struct {
	ID 			uint16	`json:"id",db:"id"`
	Name		string 	`json:"name",db:"name"`
	CountryID	uint16	`json:"countryId",db:"country_id"`
}

type Country struct {
	ID 			uint8	`json:"id",db:"id"`
	Name		string 	`json:"name",db:"name"`
	SortName	string	`json:"sortname",db:"sortname"`
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

	// Deal
	api.HandleFunc("/deals", GetDeals).Methods("GET")
	api.HandleFunc("/deal/{id}", use(GetDeal, auth)).Methods("GET")
	api.HandleFunc("/deal/{id}", use(PostDeal, auth)).Methods("POST")
	api.HandleFunc("/deal/{id}", use(DeleteDeal, auth)).Methods("DELETE")

	// User
	api.HandleFunc("/register", CreateUser).Methods("POST")
	api.HandleFunc("/login", LoginUser).Methods("POST")
	api.HandleFunc("/logout", use(LogoutUser, auth)).Methods("POST")

	fmt.Printf("listening on %d\n", conf.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), api))
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

// Decorate the request handler with Middleware
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
	values := r.URL.Query()
	var filterStrings []string
	searchText := values.Get("search_text")
	after := values.Get("after")
	before := values.Get("before")
	isPaginating := len(before) > 0 && len(after) > 0
	if isPaginating && (after == "" || before == "") {
		http.Error(w, "`before` or `after` is missing", http.StatusBadRequest)
		return
	}
	iso8601Layout := "2006-01-02T15:04:05Z"
	beforeT, _ := time.Parse(iso8601Layout, before)
	afterT, _ := time.Parse(iso8601Layout, after)
	if beforeT.After(afterT) {
		http.Error(w, "`before` is later than `after`", http.StatusBadRequest)
		return
	}

	cityId, err := strconv.Atoi(values.Get("city_id"))
	if err != nil {
		http.Error(w, "No valid city id", http.StatusBadRequest)
		return
	}

	categoryId, err := strconv.Atoi(values.Get("category_id"))
	if err == nil {
		categoryFilter := fmt.Sprintf("category_id = %d", categoryId)
		filterStrings = append(filterStrings, categoryFilter)
	}

	radiusKm, err := strconv.Atoi(values.Get("radius_km"))
	if err != nil {
		radiusKm = 10
	}
	lat, err := strconv.ParseFloat(values.Get("lat"), 64)
	lng, err := strconv.ParseFloat(values.Get("lng"), 64)
	if err == nil {
		geogColName := "point"
		distanceFilter := fmt.Sprintf("ST_Distance(%s, ST_MakePoint(%f,%f)::geography) <= %d * 1000",
			geogColName, lng, lat, radiusKm)
		filterStrings = append(filterStrings, distanceFilter)
	}

	showInactive, err := strconv.ParseBool(values.Get("show_inactive"))
	hideInactiveStr := "inactive_at IS NULL"

	if err == nil && showInactive {
		hideInactiveStr = ""
		filterStrings = append(filterStrings, hideInactiveStr)
	}


	// static options
	pageSize := 30
	postedAtColName := "posted_at"
	cityIdColName := "city_id"

	selectCols := `SELECT title, description, url_alias, 
		latitude, longitude, location_text, 
		expected_price, category_id, poster_id, 
		posted_at, updated_at, inactive_at, city_id FROM deals`

	// NOTE: Ensure no user-defined strings are in query

	cityIdFilter := fmt.Sprintf("%s = %d ", cityIdColName, cityId)
	filterStrings = append(filterStrings, cityIdFilter)

	var deals []Deal
	var rows *sql.Rows
	filterStr := ""
	dateFilter := ""

	if isPaginating {
		// Get date filter string, after most recent, or between before least recent and after floor.
		dateFilter = fmt.Sprintf("(%s > $2 OR %s < $3)", postedAtColName, postedAtColName)
		filterStrings = append(filterStrings, dateFilter)
	}
	if searchText == "" {
		http.Error(w, "No search text", http.StatusInternalServerError)
		return
	}
	titleFuzzyFilter := "title % $1"
	filterStrings = append(filterStrings, titleFuzzyFilter)
	filterStr = " WHERE " + strings.Join(filterStrings, " AND ")
	orderByStr := fmt.Sprintf("ORDER BY %s DESC", postedAtColName)
	limitStr := fmt.Sprintf("LIMIT %d", pageSize)
	query := selectCols + strings.Join([]string{filterStr, orderByStr, limitStr}, " ")
	if isPaginating {
		rows, err = db.Query(query, searchText, afterT, beforeT)
	} else {
		rows, err = db.Query(query, searchText)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	for rows.Next() {
		var deal Deal
		err = rows.Scan(&deal.Title, &deal.Description, &deal.URLAlias,
			&deal.Latitude, &deal.Longitude, &deal.LocationText,
			&deal.ExpectedPrice, &deal.CategoryID, &deal.PosterID,
			&deal.PostedAt, &deal.UpdatedAt, &deal.InactiveAt, &deal.CityID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		deals = append(deals, deal)
	}
	dealArr, err := json.Marshal(deals)
	if string(dealArr) == "null" {
		dealArr = []byte("[]")
	}
	// set struct to pointer to omit on empty
	// e.g. InactiveAt	 *time.Time  `json:"inactiveAt,omitempty",db:"inactive_at"`
	if err != nil {
		http.Error(w, "Can't marshal deals.", http.StatusInternalServerError)
	} else {
		w.Write(dealArr)
	}
}


func GetDeal(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "The cake is a lie!")
}

func PostDeal(w http.ResponseWriter, r *http.Request) {

}

func UpdateDeal(w http.ResponseWriter, r *http.Request) {

}

func DeleteDeal(w http.ResponseWriter, r *http.Request) {

}

func JoinDeal(w http.ResponseWriter, r *http.Request) {

}

func AddImageToDeal(w http.ResponseWriter, r *http.Request) {

}

func VoteOnDeal(w http.ResponseWriter, r *http.Request) {

}

func CommentOnDeal(w http.ResponseWriter, r *http.Request) {

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
	urlPostfix, err := shortid.Generate()
	passwordDigest, err := HashPassword(creds.Password)
	_, err = db.Query("insert into " +
		"users (email, password_digest, display_name, url_alias) " +
		"values ($1, $2, $3, $4)",
		creds.Email, passwordDigest, creds.DisplayName, urlPostfix)
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
