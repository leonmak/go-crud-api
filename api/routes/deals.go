package routes

import (
	"net/http"
	"time"
	"fmt"
	"groupbuying.online/utils"
	"groupbuying.online/api/structs"
	"strconv"
	"database/sql"
	"strings"
	"encoding/json"
	"github.com/iancoleman/strcase"
	"github.com/gorilla/mux"
	"errors"
	"net/url"
	"groupbuying.online/api/env"
)

func getDeals(w http.ResponseWriter, r *http.Request) {
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
			"ST_Distance(%s, %s) <= %d * 1000",
			geogColName, utils.MakePointString(lng, lat), radiusKm)
		filterStrings = append(filterStrings, distanceFilter)
	}

	showInactive, err := strconv.ParseBool(values.Get("show_inactive"))
	hideInactiveStr := "inactive_at IS NULL"
	if err == nil && showInactive {
		hideInactiveStr = ""
	}
	filterStrings = append(filterStrings, hideInactiveStr)

	selectCols := `SELECT id, title, description, thumbnail_id,
		latitude, longitude, location_text, 
		total_price, total_savings, quantity, 
		category_id, poster_id, posted_at, 
		updated_at, inactive_at, city_id FROM deals`

	var deals []structs.Deal
	var rows *sql.Rows

	// NOTE: Ensure all user-defined strings are in query parameters

	filterStr = " WHERE " + strings.Join(filterStrings, " AND ")
	orderByStr := fmt.Sprintf("ORDER BY %s DESC", postedAtColName)
	limitStr := fmt.Sprintf("LIMIT %d", pageSize)
	query := selectCols + strings.Join([]string{filterStr, orderByStr, limitStr}, " ")

	rows, err = env.Db.Query(query, queryParams...)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	for rows.Next() {
		var deal structs.Deal
		err = rows.Scan(&deal.ID, &deal.Title, &deal.Description, &deal.ThumbnailID,
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
	var deal structs.Deal
	err = env.Db.QueryRow(query, dealId).Scan(
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

func postDeal(w http.ResponseWriter, r *http.Request) {
	// On deal submit in client:
	// 1. Upload images on client side, get imageUrls and include in "images" key in payload
	// 2. Insert deal in to deals to get dealId
	// 3. Insert deal_memberships for op
	// 4. Insert deal_images for imageUrls
	// 5. Update deal thumbnail id to be first imageUrl received
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	colValues := make(map[string]interface{})
	ok := true
	var val interface{}
	var imageURLs []string
	for key, value := range result {
		snakeKey := strcase.ToSnake(key)
		switch key {
		case "title": fallthrough
		case "description": fallthrough
		case "posterId": ;fallthrough
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
		case "images":
			switch value := value.(type) {
			case []interface{}:
				for _, urlStr := range value {
					urlStrs := urlStr.(string)
					if urlStrs != "" {
						imageURLs = append(imageURLs, strings.TrimSpace(urlStrs))
					}
				}
			}

		default:
			http.Error(w, fmt.Sprintf("Invalid key '%s'", key), http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, fmt.Sprintf("Invalid value '%s'", val), http.StatusBadRequest)
			return
		}
	}

	// START Validations:
	// check if not null fields are all present
	reqCols := []string{"title", "description", "category_id", "poster_id", "city_id"}
	for _, reqCol := range reqCols {
		if colValues[reqCol] == nil {
			http.Error(w, fmt.Sprintf("Missing required field %s", reqCol), http.StatusBadRequest)
			return
		}
	}

	// check if both lat lng together, convert to sql format
	hasLat := colValues["latitude"] != nil
	hasLng := colValues["longitude"] != nil
	if (hasLat || hasLng) && hasLat != hasLng {
		http.Error(w,"Missing lat or lng", http.StatusBadRequest)
		return
	}

	posterId := colValues["poster_id"].(string)
	if !utils.IsValidUUID(posterId) {
		http.Error(w, "invalid user id", http.StatusBadRequest)
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
		valuePlaceholderStr += fmt.Sprintf(",%s", utils.MakePointString(
			colValues["latitude"], colValues["longitude"]))
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert images
	var thumbnailImageId string
	for i, imageURL := range imageURLs {
		var dealImageId string
		err = env.Db.QueryRow(
			"INSERT into deal_images (deal_id, image_url, poster_id) VALUES ($1, $2, $3) RETURNING id;",
			dealId, imageURL, posterId).Scan(&dealImageId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if i == 0  {
			thumbnailImageId = dealImageId
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Insert membership
	var membershipId string
	err = env.Db.QueryRow(
		"INSERT INTO deal_memberships (user_id, deal_id) VALUES ($1, $2) RETURNING id",
		posterId, dealId).Scan(&membershipId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = env.Db.QueryRow("UPDATE deals SET thumbnail_id=$1 WHERE id=$2 RETURNING id",
		thumbnailImageId, dealId).Scan(&dealId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write([]byte(dealId))
	}
}


func handleDeal(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet: GetDeal(w, r)
	case http.MethodPut: UpdateDeal(w, r)
	case http.MethodDelete: SetInactiveDeal(w, r)
	default: http.Error(w, fmt.Sprintf("Method not supported %s", r.Method), http.StatusBadRequest)
	}
}

func UpdateDeal(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, "no deal id found", http.StatusBadRequest)
		return
	}
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var queryValues []interface{}
	ok := true
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
		updateStrings[i] = fmt.Sprintf("%s=$%d", col, i+1)
		queryValues = append(queryValues, val)
		i++
	}
	updateStr := strings.Join(updateStrings, ",")

	// manually add because placeholder does not validly parse brackets
	hasLat := colValues["latitude"] != nil
	hasLng := colValues["longitude"] != nil
	if (hasLat || hasLng) && hasLat != hasLng {
		http.Error(w,"Missing lat or lng", http.StatusBadRequest)
		return
	}
	if hasLat && hasLng {
		updateStr += fmt.Sprintf(",point=%s", utils.MakePointString(colValues["latitude"], colValues["longitude"]))
	}

	query := fmt.Sprintf(`UPDATE deals SET %s WHERE id = $%d RETURNING id`, updateStr, len(colValues)+1)
	queryValues = append(queryValues, dealId)
	var dealIdReturned string
	err = env.Db.QueryRow(query, queryValues...).Scan(&dealIdReturned)
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
	_, err = env.Db.Query(`UPDATE deals SET inactive_at = $1 WHERE id = $2`, time.Now(), dealId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getDealMembersByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var dealMembers []structs.DealMembership
	rows, err := env.Db.Query(`SELECT u.id, u.display_name, u.image_url, deal_id, joined_at
		FROM users u 
		INNER JOIN deal_memberships m on u.id = m.user_id
		WHERE left_at ISNULL AND m.deal_id = $1`, dealId)
	defer rows.Close()
	for rows.Next() {
		var member structs.DealMembership
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

func handleDealMembership(w http.ResponseWriter, r *http.Request) {
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
	err = env.Db.QueryRow(`INSERT 
		INTO deal_memberships(user_id, deal_id, joined_at) 
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT deal_memberships_user_id_deal_id_key DO NOTHING
		RETURNING id`, userId, dealId, time.Now()).Scan(&dealMembershipId)
	return dealMembershipId, err
}

func LeaveDeal(dealId string, userId string) (dealMembershipId string, err error) {
	err = env.Db.QueryRow(`UPDATE  
		deal_memberships SET left_at = $3 
		WHERE user_id = $1 AND deal_id = $2
		RETURNING id`, userId, dealId, time.Now()).Scan(&dealMembershipId)
	return dealMembershipId, err
}

func getDealImageUrlsByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var imageUrls []string
	rows, err := env.Db.Query(`SELECT image_url from deal_images WHERE deal_id = $1`, dealId)
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

func handleDealImage(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
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
		err = env.Db.QueryRow("INSERT INTO deal_images(deal_id, poster_id, image_url) VALUES($1, $2, $3)",
			dealId, posterId, imageUrl).Scan(&dealImageId)
	case http.MethodDelete:
		dealImageId := result["dealImageId"].(string)
		if !utils.IsValidUUID(dealImageId) {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		err = env.Db.QueryRow("UPDATE deal_images SET removed_at = $1", time.Now).Scan(&dealImageId)
	default: http.Error(w, fmt.Sprintf("Method not supported %s", r.Method), http.StatusBadRequest)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Update deal image"))
}

func getDealLikeSummaryByDealId(w http.ResponseWriter, r *http.Request) {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, "invalid json", http.StatusUnprocessableEntity)
	}
	w.Write(resStr)
}

func handleDealLike(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
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
		err = env.Db.QueryRow(`INSERT INTO deal_likes(user_id, deal_id, is_upvote)
			VALUES($1, $2, $3)
			ON CONFLICT ON CONSTRAINT deal_likes_user_id_deal_id_key DO UPDATE SET is_upvote = $3
			RETURNING id`, userId, dealId, upVote).Scan(&dealId)
	case http.MethodDelete:
		err = env.Db.QueryRow(`UPDATE deal_likes SET is_upvote = NULL 
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

func getDealCommentsByDealId(w http.ResponseWriter, r *http.Request)  {
	dealId, err := getURLParamUUID("dealId", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var dealComments []structs.DealComment
	rows, err := env.Db.Query(`SELECT deal_id, user_id, u.display_name as user_name, comment_str, posted_at 
 			FROM deal_comments d
 			INNER JOIN users u ON u.id = d.user_id 
			WHERE removed_at ISNULL AND deal_id = $1`, dealId)
	defer rows.Close()
	for rows.Next() {
		var dealComment structs.DealComment
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

func handleDealComment(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
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
		err = env.Db.QueryRow(`INSERT INTO deal_comments(user_id, deal_id, comment_str) 
			VALUES($1, $2, $3)
			RETURNING id`,
			userId, dealId, comment).Scan(&dealCommentId)
	case http.MethodPut:
		err = env.Db.QueryRow(`UPDATE deal_comments SET comment_str = $1 WHERE user_id = $2 RETURNING id`,
			comment, userId).Scan(dealCommentId)
	case http.MethodDelete:
		err = env.Db.QueryRow(`UPDATE deal_comments SET removed_at = $1 RETURNING id`,
			time.Now()).Scan(&dealCommentId)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprintf("Updated user '%s' comment for deal '%s'", userId, dealId)))
}
