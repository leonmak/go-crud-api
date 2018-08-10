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
	"groupbuying.online/utils"
	"strconv"
	"strings"
	"io/ioutil"
	"github.com/iancoleman/strcase"
	"errors"
)

// TODO: Implement:
// TODO: - csrf, rate limit middleware
// TODO: - cloudinary, email verification, change name/password

// Maps to Users table
type User struct {
	ID				string 		`json:"id",db:"id"`
	DisplayName 	string		`json:"displayName",db:"display_name"`
	ImageURL		string 		`json:"imageUrl",db:"image_url"`
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
	// pointer for possible nil values
	// first image in upload is thumbnailID
	ThumbnailID		*string 	`json:"thumbnailId,omitempty",db:"thumbnail_id"`
	// location fields can be derived from lat lng (drop in) or text (reverse geocode) on POST
	Latitude		*float64	`json:"latitude,omitempty",db:"latitude"`
	Longitude		*float64	`json:"longitude,omitempty",db:"longitude"`
	// exact location text, open in maps
	LocationText	*string 	`json:"locationText,omitempty",db:"location_text"`
	TotalPrice		*float32	`json:"totalPrice,omitempty",db:"total_price"`
	TotalSavings	*float32	`json:"totalSavings,omitempty",db:"total_savings"`
	Quantity		*uint		`json:"quantity,omitempty",db:"quantity"`
	CategoryID		uint		`json:"categoryId",db:"category_id"`
	PosterID		string		`json:"posterId",db:"poster_id"`
	PostedAt		time.Time	`json:"postedAt",db:"posted_at"`
	UpdatedAt		*time.Time	`json:"updatedAt,omitempty",db:"updated_at"`
	InactiveAt		*time.Time	`json:"inactiveAt,omitempty",db:"inactive_at"`
	CityID			uint		`json:"cityId",db:"city_id"`
}

type DealCategory struct {
	ID				uint 	`json:"id",db:"id"`
	Name 			string 	`json:"name",db:"name"`
	MaxImages		uint	`json:"maxImages",db:"max_images"`
	MaxActiveDays	uint	`json:"maxActiveDays",db:"max_active_days"`
}

type DealMembership struct {
	User		User		`json:"user"`
	DealID		string		`json:"dealId",db:"deal_id"`
	JoinedAt	time.Time	`json:"joinedAt",db:"joined_at"`
	LeftAt		*time.Time	`json:"leftAt,omitEmpty",db:"left_at"`
}

type DealImage struct {
	ImageURL	url.URL		`json:"imageUrl",db:"image_url"`
	PosterID	string		`json:"posterId",db:"poster_id"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}

type DealLikes struct {
	ID			string 		`json:"id"`
	DealID		string		`json:"dealId",db:"deal_id"`
	UserID		string		`json:"userId",db:"user_id"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
	IsUpVote	bool		`json:"isUpvote",db:"is_upvote"`
}

type DealComment struct {
	Username	string 		`json:"username"`
	DealID		string		`json:"dealId",db:"deal_id"`
	UserID		string 		`json:"userId",db:"user_id"`
	Comment		string		`json:"comment",db:"comment"`
	PostedAt	time.Time	`json:"postedAt",db:"posted_at"`
}

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

type unstructuredJSON = map[string]interface{}

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
	api.HandleFunc("/deals", GetDeals).Methods(http.MethodGet)
	api.HandleFunc("/deals", use(PostDeal, auth)).Methods(http.MethodPost)
	api.HandleFunc("/deal/{dealId}", use(HandleDeal, auth)).Methods(
		http.MethodGet, http.MethodPut, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/memberships", GetDealMembersByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/membership/{userId}", use(HandleDealMembership, auth)).Methods(
		http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/likes", GetDealLikeSummaryByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/like/{userId}", use(HandleDealLike, auth)).Methods(
		http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/images", GetDealImageUrlsByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/image/{userId}", use(HandleDealImage, auth)).Methods(
		http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/comments", GetDealCommentsByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/comment/{userId}", use(HandleDealComment, auth)).Methods(
		http.MethodPost, http.MethodDelete)

	// User
	// TODO: Get another user's profile stats
	api.HandleFunc("/register", CreateUser).Methods(http.MethodPost)
	api.HandleFunc("/login", LoginUser).Methods(http.MethodPost)
	api.HandleFunc("/logout", use(LogoutUser, auth)).Methods(http.MethodPost)

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
	// static options
	pageSize := 30
	postedAtColName := "posted_at"

	values := r.URL.Query()
	var filterStrings []string

	searchText := values.Get("search_text")
	var queryParams []interface{}
	filterStr := ""
	if searchText == "" {
		http.Error(w, "No search text", http.StatusInternalServerError)
		return
	} else {
		titleFuzzyFilter := "title % $1"
		filterStrings = append(filterStrings, titleFuzzyFilter)
		queryParams = append(queryParams, searchText)
	}

	dateFilter := ""
	after := values.Get("after")
	before := values.Get("before")
	hasAfter := after != ""
	hasBefore := before != ""
	iso8601Layout := "2006-01-02T15:04:05Z"
	beforeT, err := time.Parse(iso8601Layout, before)
	afterT, err := time.Parse(iso8601Layout, after)
	if hasAfter || hasBefore {
		if hasAfter != hasBefore {
			http.Error(w, "`before` or `after` is missing", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if beforeT.After(afterT) {
			http.Error(w, "`before` is later than `after`", http.StatusBadRequest)
			return
		}
		// Get date filter string, after most recent, or between before least recent and after floor.
		dateFilter = fmt.Sprintf("(%s > $2 OR %s < $3)", postedAtColName, postedAtColName)
		filterStrings = append(filterStrings, dateFilter)
		queryParams = append(queryParams, afterT, beforeT)
	}

	posterId := values.Get("poster_id")
	if posterId != "" {
		if !utils.IsValidUUID(posterId) {
			http.Error(w, "invalid poster id", http.StatusBadRequest)
			return
		}
		posterIdColName := "poster_id"
		posterIdFilter := fmt.Sprintf("%s = %s ", posterIdColName, posterId)
		filterStrings = append(filterStrings, posterIdFilter)
	}

	cityId, err := strconv.Atoi(values.Get("city_id"))
	if err != nil {
		http.Error(w, "No valid city id", http.StatusBadRequest)
		return
	} else {
		cityIdColName := "city_id"
		cityIdFilter := fmt.Sprintf("%s = %d ", cityIdColName, cityId)
		filterStrings = append(filterStrings, cityIdFilter)
	}

	categoryId, err := strconv.Atoi(values.Get("category_id"))
	if err == nil {
		categoryFilter := fmt.Sprintf("category_id = %d", categoryId)
		filterStrings = append(filterStrings, categoryFilter)
	}

	radiusStr := values.Get("radius_km")
	latStr, lngStr := values.Get("latitude"), values.Get("longitude")
	radiusKm, errRadius := strconv.Atoi(radiusStr)
	lat, errPoint := strconv.ParseFloat(latStr, 64)
	lng, errPoint := strconv.ParseFloat(lngStr, 64)
	hasLat := latStr != ""
	hasLng := lngStr != ""
	hasRadius := radiusStr != ""
	parseRadiusErr := hasRadius && errRadius != nil
	parsePointErr := hasLat && hasLng && errPoint != nil
	missingRadiusErr := hasLat && hasLng && !hasRadius
	missingLatLngErr := !hasLat && !hasLng && hasRadius || hasLat != hasLng
	var errStr string
	if parseRadiusErr || missingRadiusErr {
		errStr = "Invalid radius"
	}
	if parsePointErr || missingLatLngErr {
		errStr = "Invalid lat/lng"
	}
	if errStr != "" {
		http.Error(w, errStr, http.StatusBadRequest)
		return
	}
	if hasLat && hasLng && hasRadius {
		geogColName := "point"
		distanceFilter := fmt.Sprintf(
			"ST_Distance(%s, ST_MakePoint(%f,%f)::geography) <= %d * 1000",
			geogColName, lng, lat, radiusKm)
		filterStrings = append(filterStrings, distanceFilter)
	}

	showInactive, err := strconv.ParseBool(values.Get("show_inactive"))
	hideInactiveStr := "inactive_at IS NULL"

	if err == nil && showInactive {
		hideInactiveStr = ""
		filterStrings = append(filterStrings, hideInactiveStr)
	}

	selectCols := `SELECT title, description, thumbnail_id,
		latitude, longitude, location_text, 
		total_price, total_savings, quantity, 
		category_id, poster_id, posted_at, 
		updated_at, inactive_at, city_id FROM deals`

	var deals []Deal
	var rows *sql.Rows

	// NOTE: Ensure all user-defined strings are in query parameters

	filterStr = " WHERE " + strings.Join(filterStrings, " AND ")
	orderByStr := fmt.Sprintf("ORDER BY %s DESC", postedAtColName)
	limitStr := fmt.Sprintf("LIMIT %d", pageSize)
	query := selectCols + strings.Join([]string{filterStr, orderByStr, limitStr}, " ")

	rows, err = db.Query(query, queryParams...)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	for rows.Next() {
		var deal Deal
		err = rows.Scan(&deal.Title, &deal.Description, &deal.ThumbnailID,
			&deal.Latitude, &deal.Longitude, &deal.LocationText,
			&deal.TotalPrice, &deal.TotalSavings, &deal.Quantity,
			&deal.CategoryID, &deal.PosterID, &deal.PostedAt,
			&deal.UpdatedAt, &deal.InactiveAt, &deal.CityID)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write(dealArr)
	}
}

func HandleDeal(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet: GetDeal(w, r)
	case http.MethodPut: UpdateDeal(w, r)
	case http.MethodDelete: SetInactiveDeal(w, r)
	default: http.Error(w, fmt.Sprintf("Method not supported %s", r.Method), http.StatusBadRequest)
	}
}

func GetDeal(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	selectCols := `SELECT title, description, thumbnail_id, 
		latitude, longitude, location_text, 
		total_price, total_savings, quantity, 
		category_id, poster_id, posted_at, 
		updated_at, inactive_at, city_id FROM deals`

	filterStr := fmt.Sprintf(" WHERE id = $1")
	query := selectCols + filterStr
	var deal Deal
	err = db.QueryRow(query, dealId).Scan(
		&deal.Title, &deal.Description, &deal.ThumbnailID,
		&deal.Latitude, &deal.Longitude, &deal.LocationText,
		&deal.TotalPrice, &deal.TotalSavings, &deal.Quantity,
		&deal.CategoryID, &deal.PosterID, &deal.PostedAt,
		&deal.UpdatedAt, &deal.InactiveAt, &deal.CityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dealArr, err := json.Marshal(deal)
	if string(dealArr) == "null" {
		dealArr = []byte("[]")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return

	} else {
		w.Write(dealArr)
	}
}

func readUnstructuredJson(r *http.Request) (unstructuredJSON, error) {
	var result unstructuredJSON
	jsonRead, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(jsonRead), &result)
	return result, nil
}

func PostDeal(w http.ResponseWriter, r *http.Request) {
	result, err := readUnstructuredJson(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	colValues := make(map[string]interface{})
	var ok bool
	var val interface{}
	for key, value := range result {
		snakeKey := strcase.ToSnake(key)
		switch key {
		case "title": fallthrough
		case "description": fallthrough
		case "posterId": fallthrough
		case "thumbnailId": fallthrough
		case "locationText":
			val, ok = value.(string)
			colValues[snakeKey] = val
		case "latitude": fallthrough
		case "longitude": fallthrough
		case "categoryId": fallthrough
		case "cityId": fallthrough
		case "totalPrice": fallthrough
		case "totalSavings": fallthrough
		case "quantity":
			val, ok = value.(float64)
			colValues[snakeKey] = val
		default:
			http.Error(w, fmt.Sprintf("Invalid key '%s'", key), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, fmt.Sprintf("Invalid value '%s'", val), http.StatusBadRequest)
			return
		}
	}

	// check if not null fields are all present
	reqCols := []string{"title", "description", "category_id", "poster_id", "city_id"}
	for _, reqCol := range reqCols {
		if colValues[reqCol] == nil {
			http.Error(w, fmt.Sprintf("Missing required field %s", reqCol), http.StatusBadRequest)
			return
		}
	}
	var cols []string
	var vals []interface{}
	for col, val := range colValues {
		cols = append(cols, col)
		vals = append(vals, val)
	}
	hasLat := colValues["latitude"] != nil
	hasLng := colValues["longitude"] != nil
	if (hasLat || hasLng) && hasLat != hasLng {
		http.Error(w,"Missing lat or lng", http.StatusBadRequest)
		return
	}
	colsStr := strings.Join(cols, ",")
	valuePlaceholders := make([]string, len(cols))
	for i:=0; i<len(cols); i++ {
		valuePlaceholders[i] = fmt.Sprintf("$%d", i+1)
	}
	valuePlaceholderStr := strings.Join(valuePlaceholders, ",")
	if hasLat && hasLng {
		colsStr += ",point"
		valuePlaceholderStr += fmt.Sprintf(",ST_MakePoint(%.6f,%.6f)",
			colValues["latitude"], colValues["longitude"])
	}
	insertStr := fmt.Sprintf(`INSERT INTO deals (%s)`, colsStr)
	valuesStr := fmt.Sprintf(`VALUES (%s)`, valuePlaceholderStr)
	returnStr := fmt.Sprintf("RETURNING %s", "id")
	query := strings.Join([]string{insertStr, valuesStr, returnStr}, " ")
	var dealId string
	err = db.QueryRow(query, vals...).Scan(&dealId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(dealId))
}

func UpdateDeal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dealId := vars["id"]
	if dealId == "" {
		http.Error(w, "no id found", http.StatusBadRequest)
		return
	}
	result, err := readUnstructuredJson(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var values []interface{}
	var ok bool
	var val interface{}
	colValues := make(map[string]interface{})
	for key, value := range result {
		snakeKey := strcase.ToSnake(key)
		switch key {
		case "title": fallthrough
		case "description": fallthrough
		case "thumbnailId": fallthrough
		case "locationText":
			val, ok = value.(string)
			colValues[snakeKey] = val
		case "latitude": fallthrough
		case "longitude": fallthrough
		case "categoryId": fallthrough
		case "cityId": fallthrough
		case "totalPrice": fallthrough
		case "totalSavings": fallthrough
		case "quantity":
			val, ok = value.(float64)
			colValues[snakeKey] = val
		default:
			http.Error(w, fmt.Sprintf("Invalid key '%s'", key), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, fmt.Sprintf("Invalid value '%s'", val), http.StatusBadRequest)
			return
		}
	}
	colValues["updated_at"] = time.Now()
	updateStrings := make([]string, len(colValues))
	i := 0
	for col, val := range colValues {
		updateStrings[i] = fmt.Sprintf("%s = $%d", col, i+1)
		values = append(values, val)
		i++
	}
	updateStr := strings.Join(updateStrings, ",")
	query := fmt.Sprintf(`UPDATE deals SET %s WHERE id = $%d RETURNING id`, updateStr, len(colValues)+1)
	values = append(values, dealId)
	var dealIdReturned string
	err = db.QueryRow(query, values...).Scan(&dealIdReturned)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(dealIdReturned))
}

func getURLParamUUID(paramName string, r *http.Request) (string, error) {
	param, err := getURLParam(paramName, r)
	if err != nil {
		return "", err
	}
	if !utils.IsValidUUID(param) {
		return "", fmt.Errorf("invalid param name '%s", param)
	}
	return param, nil
}

func getURLParam(param string, r *http.Request) (string, error) {
	vars := mux.Vars(r)
	paramVal := vars[param]
	if paramVal == "" {
		return paramVal, errors.New("no '%s' param found")
	}
	return paramVal, nil
}

func SetInactiveDeal(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err = db.Query(`UPDATE deals SET inactive_at = $1 WHERE id = $2`, time.Now(), dealId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetDealMembersByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var dealMembers []DealMembership
	rows, err := db.Query(`SELECT u.id, u.display_name, u.image_url, deal_id, joined_at
		FROM users u 
		INNER JOIN deal_memberships m on u.id = m.user_id
		WHERE left_at ISNULL AND m.deal_id = $1`, dealId)
	defer rows.Close()
	for rows.Next() {
		var member DealMembership
		rows.Scan(&member.User.ID, &member.User.DisplayName, &member.User.ImageURL,
			&member.DealID, &member.JoinedAt)
		dealMembers = append(dealMembers, member)
	}
	membersBytes, err := json.Marshal(dealMembers)
	if err != nil {
		http.Error(w, "invalid json", http.StatusUnprocessableEntity)
		return
	}
	w.Write(membersBytes)
}

func HandleDealMembership(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	userId, err := getURLParamUUID("userId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var dealMembershipId string
	switch r.Method {
	case http.MethodPost: dealMembershipId, err = JoinDeal(dealId, userId)
	case http.MethodDelete: dealMembershipId, err = LeaveDeal(dealId, userId)
	default: http.Error(w, fmt.Sprintf("Method not supported %s", r.Method), http.StatusBadRequest)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		w.Write([]byte(fmt.Sprintf("Updated %s membership for user '%s' in deal '%s'",
			dealMembershipId, userId, dealId)))
	}
}

func JoinDeal(dealId string, userId string) (dealMembershipId string, err error) {
	err = db.QueryRow(`INSERT 
		INTO deal_memberships(user_id, deal_id, joined_at) 
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT deal_memberships_user_id_deal_id_key DO NOTHING
		RETURNING id`, userId, dealId, time.Now()).Scan(&dealMembershipId)
	return dealMembershipId, err
}

func LeaveDeal(dealId string, userId string) (dealMembershipId string, err error) {
	err = db.QueryRow(`UPDATE  
		deal_memberships SET left_at = $3 
		WHERE user_id = $1 AND deal_id = $2
		RETURNING id`, userId, dealId, time.Now()).Scan(&dealMembershipId)
	return dealMembershipId, err
}

func GetDealImageUrlsByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var imageUrls []string
	rows, err := db.Query(`SELECT image_url from deal_images WHERE deal_id = $1`, dealId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var imageUrl string
		if err := rows.Scan(&imageUrl); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		imageUrls = append(imageUrls, imageUrl)
	}
	imageURLStr, err := json.Marshal(imageUrls)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(imageURLStr)
}

func HandleDealImage(w http.ResponseWriter, r *http.Request) {
	result, err := readUnstructuredJson(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var dealImageId string
	switch r.Method {
	case http.MethodPost:
		dealId := result["dealId"].(string)
		imageUrl := result["imageUrl"].(string)
		posterId := result["posterId"].(string)
		_, err := url.Parse(imageUrl)
		if !utils.IsValidUUID(dealId) || !utils.IsValidUUID(posterId) || err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		err = db.QueryRow("INSERT INTO deal_images(deal_id, poster_id, image_url) VALUES($1, $2, $3)",
			dealId, posterId, imageUrl).Scan(&dealImageId)
	case http.MethodDelete:
		dealImageId := result["dealImageId"].(string)
		if !utils.IsValidUUID(dealImageId) {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		err = db.QueryRow("UPDATE deal_images SET removed_at = $1", time.Now).Scan(&dealImageId)
	default: http.Error(w, fmt.Sprintf("Method not supported %s", r.Method), http.StatusBadRequest)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Update deal image"))
}

func GetDealLikeSummaryByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var upVotes int
	var downVotes int
	err = db.QueryRow(`SELECT 
		count(nullif(is_upvote = true, true)),
		count(nullif(is_upvote = false, true))
		FROM deal_likes
		WHERE deal_id = $1`, dealId).Scan(&upVotes, &downVotes)
	type result struct {
		upVotes int
		downVotes int
	}
	res := &result{upVotes: upVotes, downVotes: downVotes}
	resStr, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "invalid json", http.StatusUnprocessableEntity)
	}
	w.Write(resStr)
}

func HandleDealLike(w http.ResponseWriter, r *http.Request) {
	result, err := readUnstructuredJson(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dealId, ok := result["dealId"].(string)
	userId, ok := result["userId"].(string)
	upVote, ok := result["upVote"].(bool)
	if !utils.IsValidUUID(dealId) || !utils.IsValidUUID(userId) || !upVote || !ok {
		http.Error(w, "invalid value", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPost: // upsert
		err = db.QueryRow(`INSERT INTO deal_likes(user_id, deal_id, is_upvote)
			VALUES($1, $2, $3)
			ON CONFLICT ON CONSTRAINT deal_likes_user_id_deal_id_key DO UPDATE SET is_upvote = $3
			RETURNING id`, userId, dealId, upVote).Scan(&dealId)
	case http.MethodDelete:
		err = db.QueryRow(`UPDATE deal_likes SET is_upvote = NULL 
			WHERE user_id = $1 AND deal_id = $2 RETURNING id`, userId, dealId).Scan(&dealId)
	default:
		http.Error(w, "Method not supported", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprintf("Updated user '%s' like status for deal '%s'", userId, dealId)))
}

func GetDealCommentsByDealId(w http.ResponseWriter, r *http.Request)  {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var dealComments []DealComment
	rows, err := db.Query(`SELECT deal_id, user_id, u.display_name as user_name, comment_str, posted_at 
 			FROM deal_comments d
 			INNER JOIN users u ON u.id = d.user_id 
			WHERE removed_at ISNULL AND deal_id = $1`, dealId)
	defer rows.Close()
	for rows.Next() {
		var dealComment DealComment
		err = rows.Scan(&dealComment.DealID, &dealComment.UserID, &dealComment.Username,
			&dealComment.Comment, &dealComment.PostedAt)
		dealComments = append(dealComments, dealComment)
	}
	dealCommentBytes, err := json.Marshal(dealComments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Write(dealCommentBytes)
}

func HandleDealComment(w http.ResponseWriter, r *http.Request) {
	result, err := readUnstructuredJson(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dealId, ok := result["dealId"].(string)
	userId, ok := result["userId"].(string)
	comment, ok := result["comment"].(string)
	if !utils.IsValidUUID(dealId) || !utils.IsValidUUID(userId) || !ok || len(comment) > 256 {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	var dealCommentId string
	switch r.Method {
	case http.MethodPost:
		err = db.QueryRow(`INSERT INTO deal_comments(user_id, deal_id, comment_str) 
			VALUES($1, $2, $3)
			RETURNING id`,
			userId, dealId, comment).Scan(&dealCommentId)
	case http.MethodPut:
		err = db.QueryRow(`UPDATE deal_comments SET comment_str = $1 WHERE user_id = $2 RETURNING id`,
			comment, userId).Scan(dealCommentId)
	case http.MethodDelete:
		err = db.QueryRow(`UPDATE deal_comments SET removed_at = $1 RETURNING id`,
			time.Now()).Scan(&dealCommentId)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprintf("Updated user '%s' comment for deal '%s'", userId, dealId)))
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
	_, err = db.Query("INSERT INTO " +
		"USERS (email, password_digest, display_name) " +
		"VALUES ($1, $2, $3)",
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
