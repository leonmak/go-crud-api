package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"github.com/iancoleman/strcase"
	"groupbuying.online/api/env"
	"groupbuying.online/api/structs"
	"groupbuying.online/utils"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func getDeals(w http.ResponseWriter, r *http.Request) {
	// static options
	pageSize := 30
	postedAtColName := "posted_at"

	// Collect filters for getting deals
	values := r.URL.Query()
	var filterStrings []string

	// Text filter
	var queryParams []interface{}
	colCount := 0
	filterStr := ""
	if searchText, ok := values["searchText"]; ok {
		// queryParams replace dollar placeholders ($1, $2, etc.) for db to validate
		colCount++
		filterStrings = append(filterStrings, " title % " + fmt.Sprintf("$%d", colCount))
		queryParams = append(queryParams, searchText[0])
	}

	// Date filters, <-before & after-> range query
	dateFilter := ""
	before := values.Get("before")
	after := values.Get("after")
	hasAfter := after != ""
	hasBefore := before != ""
	iso8601Layout := "2006-01-02T15:04:05Z"
	beforeT, err := time.Parse(iso8601Layout, before)
	afterT, err := time.Parse(iso8601Layout, after)
	if hasAfter || hasBefore {
		if hasAfter != hasBefore {
			utils.WriteErrorJsonResponse(w, "`before` or `after` is missing")
			return
		}
		if err != nil {
			utils.WriteErrorJsonResponse(w, err.Error())
			return
		}
		if beforeT.After(afterT) {
			utils.WriteErrorJsonResponse(w, "`before` is later than `after`")
			return
		}
		// Get date filter string, after most recent, or between before least recent and after floor.
		dateFilter = fmt.Sprintf("(d.%s > $%d OR d.%s < $%d)",
			postedAtColName, colCount+1, postedAtColName, colCount+2)
		colCount += 2
		filterStrings = append(filterStrings, dateFilter)
		queryParams = append(queryParams, afterT, beforeT)
	}

	// Posted by filter
	posterId := values.Get("posterId")
	if posterId != "" {
		if !utils.IsValidUUID(posterId) {
			utils.WriteErrorJsonResponse(w, "invalid poster id")
			return
		}
		posterIdFilter := fmt.Sprintf("poster_id = $%s ", posterId)
		filterStrings = append(filterStrings, posterIdFilter)
	}

	// Category filter
	categoryId, err := strconv.Atoi(values.Get("categoryId"))
	if err == nil {
		categoryFilter := fmt.Sprintf("category_id = %d", categoryId)
		filterStrings = append(filterStrings, categoryFilter)
	}

	// Country filter
	countryCode := values.Get("countryCode")
	if countryCode != "" && govalidator.IsISO3166Alpha2(countryCode) {
		countryCodeFilter := fmt.Sprintf("country_code = '%s'", countryCode)
		filterStrings = append(filterStrings, countryCodeFilter)
	}

	// Location filter
	radiusStr := values.Get("radiusKm")
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
		utils.WriteErrorJsonResponse(w, errStr)
		return
	}
	if hasLat && hasLng && hasRadius {
		geogColName := "point"
		distanceFilter := fmt.Sprintf(
			"ST_Distance(%s, %s) <= %d * 1000",
			geogColName, utils.MakePointString(lng, lat), radiusKm)
		filterStrings = append(filterStrings, distanceFilter)
	}

	// Inactive filter
	showInactive, err := strconv.ParseBool(values.Get("showInactive"))
	hideInactiveStr := ""
	if err == nil && showInactive {
		hideInactiveStr = "inactive_at IS NOT NULL"
		filterStrings = append(filterStrings, hideInactiveStr)
	}

	selectCols := `SELECT d.id, d.title, d.description, i.image_url,
		d.latitude, d.longitude, d.location_text, 
		d.total_price, d.quantity, d.benefits,
		d.category_id, d.poster_id, d.posted_at, 
		d.updated_at, d.inactive_at FROM deals d LEFT JOIN deal_images i on d.id=i.deal_id`

	// Join on member
	memberId := values.Get("memberId")
	if memberId != "" {
		selectCols += " LEFT JOIN deal_memberships m on d.id = m.deal_id"
		colCount++
		filterStrings = append(filterStrings, fmt.Sprintf("m.user_id = $%d", colCount))
		queryParams = append(queryParams, memberId)
	}

	var deals []structs.Deal
	var rows *sql.Rows

	// NOTE: Ensure all user-defined strings are in query parameters
	filterStr = " WHERE " + strings.Join(filterStrings, " AND ")
	orderByStr := fmt.Sprintf("ORDER BY d.%s DESC", postedAtColName)
	limitStr := fmt.Sprintf("LIMIT %d", pageSize)
	query := selectCols + strings.Join([]string{filterStr, orderByStr, limitStr}, " ")

	rows, err = env.Db.Query(query, queryParams...)

	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}

	defer rows.Close()
	for rows.Next() {
		var deal structs.Deal
		err = rows.Scan(&deal.ID, &deal.Title, &deal.Description, &deal.ThumbnailUrl,
			&deal.Latitude, &deal.Longitude, &deal.LocationText,
			&deal.TotalPrice, &deal.Quantity, &deal.Benefits,
			&deal.CategoryID, &deal.PosterID, &deal.PostedAt,
			&deal.UpdatedAt, &deal.InactiveAt)
		if err != nil {
			utils.WriteErrorJsonResponse(w, err.Error())
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
		utils.WriteErrorJsonResponse(w, err.Error())
	} else {
		w.Write(dealArr)
	}
}


func GetDeal(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}

	selectCols := `SELECT title, description, thumbnail_id, 
		latitude, longitude, location_text, 
		total_price, quantity, benefits, 
		category_id, poster_id, posted_at, 
		updated_at, inactive_at FROM deals`

	filterStr := fmt.Sprintf(" WHERE id = $1")
	query := selectCols + filterStr
	var deal structs.Deal
	err = env.Db.QueryRow(query, dealId).Scan(
		&deal.Title, &deal.Description, &deal.ThumbnailUrl,
		&deal.Latitude, &deal.Longitude, &deal.LocationText,
		&deal.TotalPrice, &deal.Quantity, &deal.Benefits,
		&deal.CategoryID, &deal.PosterID, &deal.PostedAt,
		&deal.UpdatedAt, &deal.InactiveAt)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	dealArr, err := json.Marshal(deal)
	if string(dealArr) == "null" {
		dealArr = []byte("[]")
	}
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	} else {
		w.Write(dealArr)
	}
}

func getDealCategories(w http.ResponseWriter, r *http.Request) {
	var categories []structs.DealCategory
	var rows *sql.Rows
	rows, err := env.Db.Query(`SELECT id, name, display_name from deal_categories`)
	for rows.Next() {
		var category structs.DealCategory
		err = rows.Scan(&category.ID, &category.Name, &category.DisplayName)
		categories = append(categories, category)
	}
	if err != nil {
		return
	}
	utils.WriteJsonResponse(w, "categories", categories)
}


func postDeal(w http.ResponseWriter, r *http.Request) {
	// On deal submit in client:
	// 1. Upload images on client side, get imageUrls and include in "images" key in payload
	// 2. Insert deal in to deals to get dealId
	// 3. Insert deal_memberships for op
	// 4. Insert deal_images for imageUrls
	// 5. Update deal thumbnail id to be first imageUrl received
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	colValues := make(map[string]interface{})
	ok := true
	var val interface{}
	var imageURL string
	for key, value := range result {
		snakeKey := strcase.ToSnake(key)
		switch key {
		case "title": fallthrough
		case "description": fallthrough
		case "posterId": fallthrough
		case "benefits": fallthrough
		case "countryCode": fallthrough
		case "locationText":
			val, ok = value.(string)
			colValues[snakeKey] = val
		case "latitude": fallthrough
		case "longitude": fallthrough
		case "categoryId": fallthrough
		case "totalPrice": fallthrough
		case "quantity":
			val, ok = value.(float64)
			colValues[snakeKey] = val
		case "imageUrl":
			imageURL, ok = value.(string)
		default:
			log.Printf("Invalid key '%s'", key)
			continue
		}
		if !ok {
			utils.WriteErrorJsonResponse(w, fmt.Sprintf("Invalid value '%s'", val))
			return
		}
	}

	// START Validations:

	// check if not null fields are all present
	reqCols := []string{"title", "description", "category_id", "poster_id", "country_code"}
	for _, reqCol := range reqCols {
		if _, hasCol := colValues[reqCol]; !hasCol {
			utils.WriteErrorJsonResponse(w, fmt.Sprintf("Missing required field %s", reqCol))
			return
		}
	}

	// check if valid country code
	countryCode, hasCode := colValues["country_code"]
	if hasCode && !govalidator.IsISO3166Alpha2(countryCode.(string)) {
		utils.WriteErrorJsonResponse(w,"Invalid country code")
		return
	}

	// check if both lat lng together, convert to sql format
	lat, hasLat := colValues["latitude"]
	lng, hasLng := colValues["longitude"]
	if hasLat && hasLng && (hasLat != hasLng) {
		utils.WriteErrorJsonResponse(w,"Missing lat or lng")
		return
	}

	posterId := colValues["poster_id"].(string)
	if !utils.IsValidUUID(posterId) {
		utils.WriteErrorJsonResponse(w, "invalid user id")
		return
	}
	// END Validations

	var cols []string
	var vals []interface{}
	for col, val := range colValues {
		cols = append(cols, col)
		vals = append(vals, val)
	}

	colsStr := strings.Join(cols, ",")
	valuePlaceholders := make([]string, len(cols))
	for i:=0; i<len(cols); i++ {
		valuePlaceholders[i] = fmt.Sprintf("$%d", i+1)
	}
	valuePlaceholderStr := strings.Join(valuePlaceholders, ",")
	if hasLat && hasLng {
		colsStr += ",point"
		valuePlaceholderStr += fmt.Sprintf(",%s",
			utils.MakePointString(lat.(float64), lng.(float64)))
	}

	// START Insertions
	// Insert deal
	insertStr := fmt.Sprintf(`INSERT INTO deals (%s)`, colsStr)
	valuesStr := fmt.Sprintf(`VALUES (%s)`, valuePlaceholderStr)
	returnStr := fmt.Sprintf("RETURNING %s", "id")
	query := strings.Join([]string{insertStr, valuesStr, returnStr}, " ")
	var dealId string
	err = env.Db.QueryRow(query, vals...).Scan(&dealId)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}

	// Insert image
	var thumbnailImageId string
	err = env.Db.QueryRow(
		"INSERT into deal_images (deal_id, image_url, poster_id) VALUES ($1, $2, $3) RETURNING id",
		dealId, imageURL, posterId).Scan(&thumbnailImageId)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}

	// Insert membership
	var membershipId string
	err = env.Db.QueryRow(
		"INSERT INTO deal_memberships (user_id, deal_id) VALUES ($1, $2) RETURNING id",
		posterId, dealId).Scan(&membershipId)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}

	// Update deal's thumbnail lid
	if thumbnailImageId != "" {
		err = env.Db.QueryRow("UPDATE deals SET thumbnail_id=$1 WHERE id=$2 RETURNING id",
			thumbnailImageId, dealId).Scan(&dealId)
	}
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
	} else {
		utils.WriteJsonResponse(w, "dealId", dealId)
	}
}


func handleDeal(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet: GetDeal(w, r)
	case http.MethodPut: UpdateDeal(w, r)
	case http.MethodDelete: SetInactiveDeal(w, r)
	default: utils.WriteErrorJsonResponse(w, fmt.Sprintf("Method not supported %s", r.Method))
	}
}

func UpdateDeal(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, "no deal id found")
		return
	}
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	var queryValues []interface{}
	ok := true
	var val interface{}
	colValues := make(map[string]interface{})
	var imageURL string
	var posterId string
	for key, value := range result {
		snakeKey := strcase.ToSnake(key)
		switch key {
		case "title": fallthrough
		case "description": fallthrough
		case "thumbnailId": fallthrough
		case "countryCode": fallthrough
		case "benefits": fallthrough
		case "locationText":
			val, ok = value.(string)
			colValues[snakeKey] = val
		case "latitude": fallthrough
		case "longitude": fallthrough
		case "categoryId": fallthrough
		case "totalPrice": fallthrough
		case "percentDiscount": fallthrough
		case "quantity":
			val, ok = value.(float64)
			colValues[snakeKey] = val
		case "imageUrl": imageURL, ok = value.(string)
		case "posterId": posterId, ok = value.(string)
		default:
			log.Printf("Invalid key '%s'", key)
			continue
		}
		if !ok {
			utils.WriteErrorJsonResponse(w, fmt.Sprintf("Invalid value '%s'", val))
			return
		}
	}

	// Insert thumbnail image of deal id if doesn't exist
	var thumbnailImageId string
	err = env.Db.QueryRow("UPDATE deal_images SET image_url=$1 WHERE (deal_id=$2 AND poster_id=$3) RETURNING id",
		imageURL, dealId, posterId).Scan(&thumbnailImageId)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}

	// Form query string
	colValues["updated_at"] = time.Now()
	updateStrings := make([]string, len(colValues))
	i := 0
	for col, val := range colValues {
		updateStrings[i] = fmt.Sprintf("%s=$%d", col, i+1)
		queryValues = append(queryValues, val)
		i++
	}
	updateStr := strings.Join(updateStrings, ",")

	// manually add because placeholder does not validly parse brackets
	lat, hasLat := colValues["latitude"]
	lng, hasLng := colValues["longitude"]
	if (hasLat || hasLng) && hasLat != hasLng {
		utils.WriteErrorJsonResponse(w,"Missing lat or lng")
		return
	}
	if hasLat && hasLng {
		updateStr += fmt.Sprintf(",point=%s", utils.MakePointString(lat.(float64), lng.(float64)))
	} else {
		updateStr += ",point=null"
	}

	// If no values sent for a column, it will be assumed to be removed and reset to NULL
	resetCols := []string{"total_price", "quantity", "benefits", "thumbnail_id", "location_text"}
	for _, colStr := range resetCols {
		if _, ok := colValues[colStr]; !ok {
			updateStr += fmt.Sprintf(",%s=null", colStr)
		}
	}

	query := fmt.Sprintf(`UPDATE deals SET %s WHERE id = $%d RETURNING id`, updateStr, len(colValues)+1)
	queryValues = append(queryValues, dealId)
	var dealIdReturned string
	err = env.Db.QueryRow(query, queryValues...).Scan(&dealIdReturned)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	utils.WriteSuccessJsonResponse(w, fmt.Sprintf("deal %s updated", dealIdReturned))
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
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	_, err = env.Db.Query(`UPDATE deals SET inactive_at = $1 WHERE id = $2`, time.Now(), dealId)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
	}
}

func getDealMembershipByUserIdDealId(w http.ResponseWriter, r *http.Request) {
	dealId, dealIdErr := getURLParamUUID("dealId", r)
	userId, userIdErr := getURLParamUUID("userId", r)
	if userIdErr != nil || dealIdErr != nil {
		utils.WriteErrorJsonResponse(w, "invalid request")
		return
	}
	userIdMember := ""
	err := env.Db.QueryRow(`SELECT u.id FROM users u INNER JOIN deal_memberships m
		ON u.id = m.user_id
		WHERE m.deal_id = $1 AND u.id = $2`, dealId, userId).Scan(&userIdMember)
	isMember := err != sql.ErrNoRows
	utils.WriteJsonResponse(w, "result", isMember)
}

func getDealMembersByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	values := r.URL.Query()
	base := values.Get("base")
	limit := values.Get("limit")
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	limitI, err := strconv.Atoi(limit)
	if err != nil {
		utils.WriteErrorJsonResponse(w, "Invalid limit")
		return
	}
	var dealMembers []structs.DealMembership
	var rows *sql.Rows
	if base != "" {
		iso8601Layout := "2006-01-02T15:04:05Z"
		baseT, err := time.Parse(iso8601Layout, base)
		if err != nil {
			utils.WriteErrorJsonResponse(w, "Wrong time")
			return
		}
		rows, err = env.Db.Query(`SELECT u.id, u.display_name, u.image_url, joined_at
		FROM users u INNER JOIN deal_memberships m 
		ON u.id = m.user_id 
		WHERE m.deal_id = $1 AND m.joined_at > $2
		ORDER BY joined_at
		LIMIT $3;
		`, dealId, baseT, limitI)
	} else {
		rows, err = env.Db.Query(`SELECT u.id, u.display_name, u.image_url, joined_at
		FROM users u INNER JOIN deal_memberships m 
		ON u.id = m.user_id 
		WHERE m.deal_id = $1 
		ORDER BY joined_at
		LIMIT $2;
		`, dealId, limitI)
	}
	defer rows.Close()
	for rows.Next() {
		var member structs.DealMembership
		rows.Scan(&member.User.ID, &member.User.DisplayName, &member.User.ImageURL, &member.JoinedAt)
		dealMembers = append(dealMembers, member)
	}
	membersBytes, err := json.Marshal(dealMembers)
	if err != nil {
		utils.WriteErrorJsonResponse(w, "invalid json")
		return
	}
	w.Write(membersBytes)
}

func handleDealMembership(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	dealId, dealIdOk := result["dealId"].(string)
	userId, userIdOk := result["userId"].(string)
	if !dealIdOk || !userIdOk {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	switch r.Method {
	case http.MethodPost:
		dealMembershipId, err := JoinDeal(dealId, userId)
		if err == nil {
			log.Print(fmt.Sprintf("Updated membership for user '%s' in deal '%s' in %s",
				userId, dealId, dealMembershipId))
			utils.WriteSuccessJsonResponse(w, "Updated membership")
		}
	case http.MethodDelete:
		err = LeaveDeal(dealId, userId)
		if err == nil {
			log.Print(fmt.Sprintf("Removed membership for user '%s' in deal '%s'", userId, dealId))
			utils.WriteSuccessJsonResponse(w, "Removed membership")
		}
	default: utils.WriteErrorJsonResponse(w, fmt.Sprintf("Method not supported %s", r.Method))
	}
	// Handle errors
	switch err {
	case sql.ErrNoRows: log.Printf("User has no membership")
	default: return
	}
	utils.WriteErrorJsonResponse(w, err.Error())
}

func JoinDeal(dealId string, userId string) (dealMembershipId string, err error) {
	// if already a member do nothing
	err = env.Db.QueryRow(`INSERT  
		INTO deal_memberships(user_id, deal_id, joined_at) 
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT deal_memberships_user_id_deal_id_key DO NOTHING 
		RETURNING id`, userId, dealId, time.Now()).Scan(&dealMembershipId)
	return dealMembershipId, err
}

func LeaveDeal(dealId string, userId string) (err error) {
	_, err = env.Db.Query(`DELETE FROM deal_memberships 
		WHERE user_id = $1 AND deal_id = $2`, userId, dealId)
	return err
}

func getDealImageUrlsByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	var imageUrls []string
	rows, err := env.Db.Query(`SELECT image_url from deal_images WHERE deal_id = $1`, dealId)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var imageUrl string
		if err := rows.Scan(&imageUrl); err != nil {
			utils.WriteErrorJsonResponse(w, err.Error())
			return
		}
		imageUrls = append(imageUrls, imageUrl)
	}
	imageURLStr, err := json.Marshal(imageUrls)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	w.Write(imageURLStr)
}

func handleDealImage(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
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
			utils.WriteErrorJsonResponse(w, "invalid id")
			return
		}
		err = env.Db.QueryRow("INSERT INTO deal_images(deal_id, poster_id, image_url) VALUES($1, $2, $3)",
			dealId, posterId, imageUrl).Scan(&dealImageId)
	case http.MethodDelete:
		dealImageId := result["dealImageId"].(string)
		if !utils.IsValidUUID(dealImageId) {
			utils.WriteErrorJsonResponse(w, "invalid id")
			return
		}
		err = env.Db.QueryRow("UPDATE deal_images SET removed_at = $1", time.Now).Scan(&dealImageId)
	default: utils.WriteErrorJsonResponse(w, fmt.Sprintf("Method not supported %s", r.Method))
	}
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	utils.WriteJsonResponse(w, "result", "Updated deal image")
}

func getDealLikeByUserId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	userId, err := getURLParamUUID("userId", r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	isUpvote := false
	env.Db.QueryRow("SELECT is_upvote from deal_likes WHERE deal_id=$1 AND user_id=$2",
		dealId, userId).Scan(&isUpvote)
	utils.WriteJsonResponse(w, "isUpvote", isUpvote)
}

func getDealLikeSummaryByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	var upVotes int
	var downVotes int
	err = env.Db.QueryRow(`SELECT 
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
		utils.WriteErrorJsonResponse(w, "invalid json")
	}
	w.Write(resStr)
}

func handleDealLike(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	dealId, ok := result["dealId"].(string)
	userId, ok := result["userId"].(string)
	upVote, ok := result["upVote"].(bool)
	if !utils.IsValidUUID(dealId) || !utils.IsValidUUID(userId) || !ok {
		utils.WriteErrorJsonResponse(w, "invalid value")
		return
	}
	switch r.Method {
	case http.MethodPost: // upsert
		err = env.Db.QueryRow(`INSERT INTO deal_likes(user_id, deal_id, is_upvote)
			VALUES($1, $2, $3)
			ON CONFLICT ON CONSTRAINT deal_likes_user_id_deal_id_key DO UPDATE SET is_upvote = $3
			RETURNING id`, userId, dealId, upVote).Scan(&dealId)
	case http.MethodDelete:
		err = env.Db.QueryRow(`UPDATE deal_likes SET is_upvote = NULL 
			WHERE user_id = $1 AND deal_id = $2 RETURNING id`, userId, dealId).Scan(&dealId)
	default:
		utils.WriteErrorJsonResponse(w, "Method not supported")
		return
	}
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	utils.WriteJsonResponse(w, "result",
		fmt.Sprintf("Updated user '%s' like status for deal '%s'", userId, dealId))
}

func getDealCommentsByDealId(w http.ResponseWriter, r *http.Request)  {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	var dealComments []structs.DealComment
	rows, err := env.Db.Query(`SELECT d.id, d.deal_id, d.user_id, u.display_name, d.comment_str, d.posted_at 
 			FROM deal_comments d
 			INNER JOIN users u ON u.id = d.user_id 
			WHERE removed_at ISNULL AND deal_id = $1`, dealId)
	defer rows.Close()
	for rows.Next() {
		var dealComment structs.DealComment
		err = rows.Scan(&dealComment.ID, &dealComment.DealID, &dealComment.UserID, &dealComment.Username,
			&dealComment.Comment, &dealComment.PostedAt)
		dealComments = append(dealComments, dealComment)
	}
	dealCommentBytes, err := json.Marshal(dealComments)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	w.Write(dealCommentBytes)
}

func handleDealComment(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	dealId, ok := result["dealId"].(string)
	userId, ok := result["userId"].(string)
	comment, ok := result["comment"].(string)
	if !utils.IsValidUUID(dealId) || !utils.IsValidUUID(userId) || !ok || len(comment) > 240 {
		utils.WriteErrorJsonResponse(w, "invalid input")
		return
	}
	id, ok := result["id"].(string)
	if !ok && r.Method != http.MethodPost {
		utils.WriteErrorJsonResponse(w, "invalid input")
		return
	}
	var dealCommentId string
	switch r.Method {
	case http.MethodPost:
		err = env.Db.QueryRow(`INSERT INTO deal_comments(user_id, deal_id, comment_str) 
			VALUES($1, $2, $3)
			RETURNING id`,
			userId, dealId, comment).Scan(&dealCommentId)
	case http.MethodPut:
		err = env.Db.QueryRow(`UPDATE deal_comments SET comment_str = $1 WHERE id = $2 RETURNING id`,
			comment, id).Scan(&dealCommentId)
	case http.MethodDelete:
		err = env.Db.QueryRow(`UPDATE deal_comments SET removed_at = $1 WHERE id=$2 RETURNING id`,
			time.Now(), id).Scan(&dealCommentId)
	}
	if err != nil {
		utils.WriteErrorJsonResponse(w, err.Error())
		return
	}
	utils.WriteJsonResponse(w, "result",
		fmt.Sprintf("Updated user '%s' comment for deal '%s'", userId, dealId))
}
