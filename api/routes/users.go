package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/asaskevich/govalidator"
	"groupbuying.online/api/env"
	"groupbuying.online/api/structs"
	"groupbuying.online/api/utils"
	"net/http"
	"strings"
	"time"
)

// Used when getting other users, response does not contain auth info
func getUserById(w http.ResponseWriter, r *http.Request) {
	userId, err := getURLParamUUID("userId", r)
	user := structs.User{ID: userId}
	if err != nil || userId == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	err = env.Db.QueryRow("SELECT image_url, display_name, country_code, fir_id FROM users WHERE id=$1",
		userId).Scan(&user.ImageURL, &user.DisplayName, &user.CountryCode, &user.FIRID)
	if err != nil {
		utils.WriteError(w, "user not found")
		return
	}
	userBytes, err := json.Marshal(user)
	utils.CheckFatalError(w, err)
	_, err = w.Write(userBytes)
	utils.CheckFatalError(w, err)
}

// Used by login methods, response includes auth info
func getUserByEmail(email string) (user structs.User, err error) {
	err = env.Db.QueryRow("SELECT id, image_url, display_name, " +
		"country_code, auth_type, email, fir_id " +
		"FROM users u " +
		"WHERE email=$1 " +
		"AND NOT EXISTS (SELECT user_id FROM users_banned u_b WHERE u_b.user_id=u.id)",
		email).Scan(
			&user.ID, &user.ImageURL, &user.DisplayName,
			&user.CountryCode, &user.AuthType, &user.Email, &user.FIRID)
	if err != nil {
		return user, fmt.Errorf("user not found")
	} else {
		return user, nil
	}
}

// Auth
func logoutUser(w http.ResponseWriter, r *http.Request) {
	session, _ := env.Store.Get(r, env.Conf.SessionName)
	session.Values["authenticated"] = false
	delete(session.Values, "userId")
	err := session.Save(r, w)
	utils.CheckFatalError(w, err)
	utils.WriteSuccessJsonResponse(w, "")
}

// Insert a new user with unverified email
func registerEmailUser(w http.ResponseWriter, r *http.Request) {
	creds := &structs.UserCredentials{}
	authType := "email"
	err := json.NewDecoder(r.Body).Decode(creds)
	utils.CheckFatalError(w, err)

	err = utils.IsValidUsername(creds.DisplayName)
	utils.CheckFatalError(w, err)

	err = verifyToken(creds.Token)
	utils.CheckFatalError(w, err)

	var userId string
	creds.Email = strings.ToLower(creds.Email)
	err = env.Db.QueryRow("INSERT INTO USERS " +
		"(email, display_name, auth_type, country_code, fir_id) " +
		"VALUES ($1, $2, $3, $4, $5) RETURNING id;",
		creds.Email, creds.DisplayName, authType, creds.CountryCode, creds.FIRID).Scan(&userId)
	utils.CheckFatalError(w, err)

	user := structs.User{
		ID: userId,
		DisplayName: creds.DisplayName,
		CountryCode: creds.CountryCode,
		AuthType: &authType,
		Email: &creds.Email,
		FIRID: creds.FIRID,
	}
	respondUser(user, w)
}

func verifyToken(idToken string) error {
	ctx := context.Background()
	client, err := env.Firebase.Auth(ctx)
	if err != nil {
		return err
	}

	_, err = client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return err
	}
	return nil
}

func loginEmailUser(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	utils.CheckFatalError(w, err)

	email := result["email"].(string)
	token := result["token"].(string)
	if email == "" || token == "" {
		w.WriteHeader(http.StatusBadRequest)
		utils.WriteError(w, "invalid input")
		return
	}
	err = verifyToken(token)
	utils.CheckFatalError(w, err)

	user, err := getUserByEmail(email)
	utils.CheckFatalError(w, err)

	// Save authenticated session if successful
	saveSession(user, w, r)
	respondUser(user, w)
}

func writeToRegisterJson(w http.ResponseWriter) {
	utils.WriteJsonResponse(w, "to_register", true)
}

func readSocialCredentials(r *http.Request) (*structs.SocialSignInCredentials, error) {
	creds := &structs.SocialSignInCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	creds.Email = strings.ToLower(creds.Email)
	return creds, err
}

func saveSession(user structs.User, w http.ResponseWriter, r *http.Request) {
	session, _ := env.Store.Get(r, env.Conf.SessionName)
	session.Values["authenticated"] = true
	session.Values["userId"] = user.ID
	err := session.Save(r, w)
	utils.CheckFatalError(w, err)
}

func respondUser(user structs.User, w http.ResponseWriter) {
	b, err := json.Marshal(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, err = w.Write(b)
	}
	utils.CheckFatalError(w, err)
}

// Google Auth
func loginGoogleUser(w http.ResponseWriter, r *http.Request) {
	creds, err := readSocialCredentials(r)
	utils.CheckFatalError(w, err)

	isValid := validateGoogleUserToken(creds.Email, creds.UserToken)
	if !isValid {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := getUserByEmail(creds.Email)
	if err == nil {
		saveSession(user, w, r)
		respondUser(user, w)
	} else {
		writeToRegisterJson(w)
	}
}

// Check if token's email matches token supplied
func validateGoogleUserToken(email string, userToken string) bool {
	validateTokenLink := fmt.Sprintf(
		"https://www.googleapis.com/oauth2/v3/tokeninfo?id_token=%s", userToken)
	resp, err := http.Get(validateTokenLink)
	if err != nil {
		return false
	}
	utils.CloseResponse(resp)
	jsonResp, err := utils.ReadResponseToJson(resp)
	isValid := jsonResp["email"].(string) == email
	return isValid
}

// Facebook Auth
func loginFacebookUser(w http.ResponseWriter, r *http.Request) {
	// checks userId, userToken from FBLoginKit,
	// and returns {"to_register": true} if valid but not registered
	// or user object if valid and registered.
	creds, err := readSocialCredentials(r)
	utils.CheckFatalError(w, err)

	appToken, err := getFacebookAppToken()
	utils.CheckFatalError(w, err)

	isValid := validateFacebookUserToken(appToken, creds.UserToken, creds.UserID)
	if !isValid {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := getUserByEmail(creds.Email)
	if err == nil {
		saveSession(user, w, r)
		respondUser(user, w)
	} else {
		writeToRegisterJson(w)
	}
}

func getFacebookAppToken() (appToken string, err error) {
	clientId := env.Conf.FBAppId
	clientSecret := env.Conf.FBAppSecret
	appLink := "https://graph.facebook.com/oauth/access_token?client_id=" + clientId +
		"&client_secret=" + clientSecret + "&grant_type=client_credentials"

	resp, err := http.Get(appLink)
	if err != nil {
		return "", fmt.Errorf("could not get FB App Token")
	}
	utils.CloseResponse(resp)
	jsonResp, err := utils.ReadResponseToJson(resp)
	appToken = jsonResp["access_token"].(string)
	return appToken, nil
}

func validateFacebookUserToken(appToken string, userToken string, userId string) (bool) {
	// Checks user token is valid and user_id in response is same as given userId
	validateTokenLink := "https://graph.facebook.com/debug_token?input_token="+ userToken +
		"&access_token=" + appToken
	resp, err := http.Get(validateTokenLink)
	if err != nil {
		return false
	}
	utils.CloseResponse(resp)
	jsonResp, err := utils.ReadResponseToJson(resp)
	jsonRespData := jsonResp["data"].(utils.UnstructuredJSON)
	isValid := jsonRespData["is_valid"].(bool) && jsonRespData["user_id"].(string) == userId
	return isValid
}

func registerBySocialMedia(w http.ResponseWriter, r *http.Request) {
	creds := &structs.UserCredentialSocialMedia{}
	err := json.NewDecoder(r.Body).Decode(creds)

	var id string
	err = env.Db.QueryRow("INSERT INTO users (email, display_name, image_url, auth_type, country_code, fir_id)" +
		" VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;",
		creds.Email, creds.DisplayName, creds.ImageUrl, creds.AuthType, creds.CountryCode, creds.FIRID).Scan(&id)
	utils.CheckFatalError(w, err)

	user := structs.User{
		ID: id,
		ImageURL: &creds.ImageUrl,
		DisplayName: creds.DisplayName,
		CountryCode: creds.CountryCode,
		AuthType: &creds.AuthType,
		Email: &creds.Email,
		FIRID: creds.FIRID,
	}
	saveSession(user, w, r)
	respondUser(user, w)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	utils.CheckFatalError(w, err)

	displayName := result["displayName"].(string)
	countryCode := result["countryCode"].(string)
	imageUrl := result["imageUrl"].(string)
	userId := result["userId"].(string)
	if displayName == "" || countryCode == ""  {
		utils.WriteErrorJsonResponse(w, "missing fields")
		return
	}
	if !govalidator.IsISO3166Alpha2(countryCode) || !utils.IsValidUUID(userId) {
		utils.WriteErrorJsonResponse(w, "invalid fields")
		return
	}
	if imageUrl != "" && !govalidator.IsURL(imageUrl) {
		utils.WriteErrorJsonResponse(w, "invalid image")
		return
	}
	query := "UPDATE users SET display_name=$1, country_code=$2"
	queryParams := []interface{}{ displayName, countryCode }
	if imageUrl != "" {
		query += ", image_url=$3"
		queryParams = append(queryParams, imageUrl)
	}
	query += fmt.Sprintf(" WHERE id='%s' RETURNING id", userId)
	err = env.Db.QueryRow(query, queryParams...).Scan(&userId)
	utils.CheckFatalError(w, err)

	utils.WriteSuccessJsonResponse(w, "updated user")
}

func blockUser(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	utils.CheckFatalError(w, err)

	blockedId, ok1 := result["blockedId"].(string)
	userId, ok2 := result["userId"].(string)
	reqUserId, ok3 := utils.GetUserIdInSession(r)

	if !utils.IsValidUUID(userId) || !ok1 || !ok2 || !ok3 || reqUserId != userId {
		utils.WriteErrorJsonResponse(w,"invalid input")
		return
	}
	var tupleId string
	switch r.Method {
	case http.MethodPost:
		err = env.Db.QueryRow(
			`INSERT INTO users_blocked (user_id, blocked_id) VALUES ($1, $2) RETURNING id`,
			userId, blockedId).Scan(&tupleId)
		utils.CheckFatalError(w, err)
		var blockedFirId string
		err = env.Db.QueryRow(`SELECT fir_id FROM users WHERE id=$1`, blockedId).Scan(&blockedFirId)
		utils.CheckFatalError(w, err)
		utils.WriteJsonResponse(w, "blockedFirId", blockedFirId)
	case http.MethodDelete:
		err = env.Db.QueryRow(
			`DELETE FROM users_blocked WHERE user_id = $1 AND blocked_id = $2 RETURNING id`,
			userId, blockedId).Scan(&tupleId)
		utils.CheckFatalError(w, err)
		utils.WriteSuccessJsonResponse(w, tupleId)
	}
}

func reportUser(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	utils.CheckFatalError(w, err)

	reportedId, ok1 := result["reportedId"].(string)
	reporterId, ok2 := result["reporterId"].(string)
	reason, ok3 := result["reason"].(string)
	reqUserId, ok4 := utils.GetUserIdInSession(r)

	if !utils.IsValidUUID(reporterId) || !ok1 || !ok2 || !ok3 || !ok4 || reqUserId != reporterId {
		utils.WriteErrorJsonResponse(w,"invalid input")
		return
	}

	var tupleId string
	err = env.Db.QueryRow(`
		INSERT INTO users_reported (reporter_id, reported_id, reason) 
		VALUES ($1, $2, $3) RETURNING id`, reporterId, reportedId, reason).Scan(&tupleId)
	utils.CheckFatalError(w, err)
	utils.WriteSuccessJsonResponse(w, tupleId)
}

func isUserBanned(w http.ResponseWriter, r *http.Request) {
	userId, ok := utils.GetUserIdInSession(r)
	if !ok || !utils.IsValidUUID(userId) {
		utils.WriteError(w,"invalid input")
		return
	}

	var banDate time.Time
	key := "isBanned"
	if err := env.Db.QueryRow(`SELECT created_at FROM users_banned WHERE user_id=$1`,
		userId).Scan(&banDate); err == sql.ErrNoRows {
		utils.WriteJsonResponse(w, key, false)
	} else {
		utils.WriteJsonResponse(w, key, true)
	}
}
