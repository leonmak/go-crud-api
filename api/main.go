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
)

// TODO: Implement:
// TODO: - csrf, rate limit middleware
// TODO: - cloudinary, email verification, change name/password
type User struct {
	ID					string 		`json:"id,omitempty",db:"id"`
	Email	 			string		`json:"email",db:"email"`
	DisplayName 		string		`json:"displayName",db:"display_name"`
	URLAlias 			string 		`json:"urlAlias",db:"url_alias"`
	PasswordDigest		string 		`json:"passwordDigest",db:"password_digest"`
	ImageURL			string 		`json:"imageUrl,omitEmpty",db:"image_url"`
	VerifyEmailSentAt	time.Time	`json:"verifyEmailSentAt,omitEmpty",db:"verify_email_sent_at"`
	IsVerified			bool		`json:"isVerified,omitEmpty",db:"is_verified"`
}

// For marshalling login / register requests
type UserCredentials struct {
	Email 		string	`json:"email"`
	Password	string	`json:"password"`
	DisplayName string	`json:"displayName,omitEmpty"`
}

type Deal struct {
	Title			string			`json:"title",db:"title"`
	Description 	string			`json:"description",db:"description"`
	URLAlias 		string 			`json:"urlAlias",db:"url_alias"`
	Lat				float64			`json:"latitude",db:"latitude"`
	Long			float64			`json:"longitude",db:"longitude"`
	LocationText	string 			`json:"locationText",db:"location_text"`
	ExpectedPrice	float32			`json:"expectedPrice",db:"expected_price"`
	CategoryID		string			`json:"categoryId",db:"category_id"`
	PosterID		string			`json:"posterId",db:"poster_id"`
	PostedAt		time.Time		`json:"postedAt",db:"posted_at"`
	UpdatedAt		time.Time		`json:"updatedAt",db:"updated_at"`
	IsInactive		bool			`json:"isInactive",db:"is_inactive"`
}

type DealCategory struct {
	ID				int8 	`json:"id",db:"id"`
	Name 			string 	`json:"name",db:"name"`
	MaxImages		int8	`json:"maxImages",db:"max_images"`
	MaxActiveDays	int8	`json:"maxActiveDays",db:"max_active_days"`
}

type DealMembership struct {
	ID			string 		`json:"id",db:"id"`
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
	values := r.URL.Query()
	after := values.Get("after")
	before := values.Get("before")
	beforeFloor := values.Get("before_floor")

	lat, err := strconv.ParseFloat(values.Get("lat"), 64)
	lng, err := strconv.ParseFloat(values.Get("lng"), 64)
	if err != nil {
		http.Error(w, "Missing lat/lng.", http.StatusBadRequest)
		return
	}

	showInactive, err := strconv.ParseBool(values.Get("show_inactive"))
	filterInactiveStr := "AND is_inactive = "
	if err == nil && showInactive {
		filterInactiveStr += "true"
	} else {
		filterInactiveStr += "false"
	}

	radiusKm, err := strconv.Atoi(values.Get("radius_km"))
	if err != nil {
		radiusKm = 10
	}

	// Get search filter string
	pageSize := 30
	geogColName := "point"
	updatedColName := "updated_at"

	// Ensure no user-defined strings are in query
	selectCols := ` SELECT title, description, url_alias, latitude, longitude, location_text, expected_price, 
		category_id, poster_id, posted_at, updated_at, is_inactive FROM deals`
	distanceFilter := fmt.Sprintf(" ST_Distance(%s, ST_MakePoint(%f,%f)::geography) <= %d * 1000",
		geogColName, lng, lat, radiusKm)
	isPaginating := len(before) > 0 && len(after) > 0

	var deals []Deal
	var rows *sql.Rows
	iso8601Layout := "2006-01-02T15:04:05Z"
	beforeFloorT, _ := time.Parse(iso8601Layout, beforeFloor)

	if isPaginating {
		// Get date filter string, after most recent
		// or between before least recent and after floor.
		query := selectCols + " WHERE" + distanceFilter +
			fmt.Sprintf(" AND (%s > $1 OR (%s BETWEEN $2 AND $3)) ", updatedColName, updatedColName) +
			filterInactiveStr + fmt.Sprintf(" ORDER BY %s DESC LIMIT %d", updatedColName, pageSize)
		rows, err = db.Query(query, after, before, beforeFloor)
	} else {
		// Get date filter string, after floor
		query := selectCols + " WHERE" + distanceFilter +
			fmt.Sprintf(" AND %s > $1 ", updatedColName) + filterInactiveStr +
			fmt.Sprintf(" ORDER BY %s DESC LIMIT %d", updatedColName, pageSize)
		rows, err = db.Query(query, beforeFloorT)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var deal Deal
		err = rows.Scan(&deal.Title, &deal.Description, &deal.URLAlias, &deal.Lat, &deal.Long, &deal.LocationText,
			&deal.ExpectedPrice, &deal.CategoryID, &deal.PosterID, &deal.PostedAt, &deal.UpdatedAt, &deal.IsInactive)
		if err != nil {
			http.Error(w, "Error scanning row.", http.StatusInternalServerError)
			return
		}
		deals = append(deals, deal)
	}
	dealArr, err := json.Marshal(deals)
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
