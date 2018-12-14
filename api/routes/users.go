package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Automattic/go-gravatar"
	"google.golang.org/appengine"
	"groupbuying.online/api/env"
	"groupbuying.online/api/structs"
	"groupbuying.online/api/utils"
	"log"
	"net/http"
	"strings"
)

// Info
func getUserById(w http.ResponseWriter, r *http.Request) {
	userId, err := getURLParamUUID("userId", r)
	user := structs.User{ID: userId}
	if err != nil || userId == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	err = env.Db.QueryRow("SELECT image_url, display_name FROM users WHERE id=$1",
		userId).Scan(&user.ImageURL, &user.DisplayName)
	if err != nil {
		utils.WriteError(w, "user not found")
	} else {
		userBytes, _ := json.Marshal(user)
		w.Write(userBytes)
	}
}

func getUserByEmail(email string) (user structs.User, err error){
	err = env.Db.QueryRow("SELECT id, image_url, display_name FROM users WHERE email=$1",
		email).Scan(&user.ID, &user.ImageURL, &user.DisplayName)
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
	session.Save(r, w)
}

func getGravatarUrl(email string) (url string) {
	g := gravatar.NewGravatarFromEmail(email)
	return g.GetURL()
}

// Insert a new user with unverified email
func registerEmailUser(w http.ResponseWriter, r *http.Request) {
	creds := &structs.UserCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		utils.WriteError(w, "invalid input")
		return
	}
	if err = utils.IsValidUsername(creds.DisplayName); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		utils.WriteError(w, err.Error())
		return
	}
	err = verifyToken(creds.Token, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		utils.WriteError(w, "invalid token")
		return
	}
	var userId string
	creds.Email = strings.ToLower(creds.Email)
	imageUrl := getGravatarUrl(creds.Email)
	err = env.Db.QueryRow("INSERT INTO USERS " +
		"(email, display_name, image_url, auth_type) " +
		"VALUES ($1, $2, $3, $4) RETURNING id;",
		creds.Email, creds.DisplayName, imageUrl, "email").Scan(&userId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		utils.WriteError(w, "user already exists")
		return
	}
	user := structs.User{ID: userId, DisplayName: creds.DisplayName, ImageURL: &imageUrl}
	respondUser(user, w)
}

func verifyToken(idToken string, r *http.Request) error {
	ctx := context.Background()
	if appengine.IsAppEngine() {
		ctx = appengine.NewContext(r)
	}
	client, err := env.Firebase.Auth(ctx)
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}
	_, err = client.VerifyIDToken(ctx, idToken)
	if err != nil {
		log.Fatalf("error verifying ID token: %v\n", err)
	}
	return err
}

func loginEmailUser(w http.ResponseWriter, r *http.Request) {
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	email := result["email"].(string)
	token := result["token"].(string)
	if email == "" || token == "" {
		w.WriteHeader(http.StatusBadRequest)
		utils.WriteError(w, "invalid input")
		return
	}
	err = verifyToken(token, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		utils.WriteError(w, "invalid token")
	}
	user := structs.User{}
	err = env.Db.QueryRow("SELECT id, image_url, display_name FROM users WHERE email=$1",
		email).Scan(&user.ID, &user.ImageURL, &user.DisplayName)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		utils.WriteError(w, "error logging in")
		return
	}
	// Save authenticated session if successful
	log.Printf("Login Successful")
	w.WriteHeader(http.StatusOK)
	saveSession(w, r)
	respondUser(user, w)
}

func writeRegisterJson(w http.ResponseWriter) {
	utils.WriteJsonResponse(w, "to_register", true)
}

func readSocialCredentials(r *http.Request) (*structs.SocialSignInCredentials, error) {
	creds := &structs.SocialSignInCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	creds.Email = strings.ToLower(creds.Email)
	return creds, err
}

func saveSession(w http.ResponseWriter, r *http.Request) {
	session, _ := env.Store.Get(r, env.Conf.SessionName)
	session.Values["authenticated"] = true
	session.Save(r, w)
}

func respondUser(user structs.User, w http.ResponseWriter) {
	b, err := json.Marshal(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.Write(b)
	}
}

// Google Auth
func loginGoogleUser(w http.ResponseWriter, r *http.Request) {
	creds, err := readSocialCredentials(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	isValid := validateGoogleUserToken(creds.Email, creds.UserToken, r)
	if !isValid {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := getUserByEmail(creds.Email)
	if err == nil {
		saveSession(w, r)
		respondUser(user, w)
	} else {
		writeRegisterJson(w)
	}
}

func validateGoogleUserToken(email string, userToken string, r *http.Request) bool {
	validateTokenLink := fmt.Sprintf(
		"https://www.googleapis.com/oauth2/v3/tokeninfo?id_token=%s", userToken)
	resp, err := http.Get(validateTokenLink)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	jsonResp := utils.ReadResponseToJson(resp)
	isValid := jsonResp["email"].(string) == email
	return isValid
}


// Facebook Auth
func loginFacebookUser(w http.ResponseWriter, r *http.Request) {
	// checks userId, userToken from FBLoginKit,
	// and returns {"to_register": true} if valid but not registered
	// or user object if valid and registered.
	creds, err := readSocialCredentials(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	appToken, err := getFacebookAppToken(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	isValid := validateFacebookUserToken(r, appToken, creds.UserToken, creds.UserID)
	if !isValid {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := getUserByEmail(creds.Email)
	if err == nil {
		saveSession(w, r)
		respondUser(user, w)
	} else {
		writeRegisterJson(w)
	}
}

func getFacebookAppToken(r *http.Request) (appToken string, err error) {
	clientId := env.Conf.FBAppId
	clientSecret := env.Conf.FBAppSecret
	appLink := "https://graph.facebook.com/oauth/access_token?client_id=" + clientId +
		"&client_secret=" + clientSecret + "&grant_type=client_credentials"

	resp, err := http.Get(appLink)
	if err != nil {
		return "", fmt.Errorf("could not get FB App Token")
	}
	defer resp.Body.Close()
	jsonResp := utils.ReadResponseToJson(resp)
	appToken = jsonResp["access_token"].(string)
	return appToken, nil
}

func validateFacebookUserToken(r *http.Request, appToken string, userToken string, userId string) (bool) {
	// Checks user token is valid and user_id in response is same as given userId
	validateTokenLink := "https://graph.facebook.com/debug_token?input_token="+ userToken +
		"&access_token=" + appToken
	resp, err := http.Get(validateTokenLink)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	jsonResp := utils.ReadResponseToJson(resp)["data"].(utils.UnstructuredJSON)
	isValid := jsonResp["is_valid"].(bool) && jsonResp["user_id"].(string) == userId
	return isValid
}

func registerBySocialMedia(w http.ResponseWriter, r *http.Request) {
	//(email string, displayName string)
	creds := &structs.UserCredentialSocialMedia{}
	err := json.NewDecoder(r.Body).Decode(creds)

	var id string
	err = env.Db.QueryRow("INSERT INTO USERS (email, display_name, image_url, auth_type) " +
		"VALUES ($1, $2, $3, $4) RETURNING id;",
		creds.Email, creds.DisplayName, creds.ImageUrl, creds.AuthType).Scan(&id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		utils.WriteError(w, "could not retrieve user")
		return
	}
	user := structs.User{ID: id, ImageURL: &creds.ImageUrl, DisplayName: creds.DisplayName}
	respondUser(user, w)
}
