package routes

import (
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"encoding/json"
	"log"
	"database/sql"
	"groupbuying.online/api/structs"
	"groupbuying.online/api/env"
	"fmt"
	"groupbuying.online/utils"
	"time"
)

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func logoutUser(w http.ResponseWriter, r *http.Request) {
	session, _ := env.Store.Get(r, env.Conf.SessionName)
	session.Values["authenticated"] = false
	session.Save(r, w)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func createUser(w http.ResponseWriter, r *http.Request) {
	creds := &structs.UserCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil || creds.DisplayName == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Invalid submission.")
		return
	}
	passwordDigest, err := HashPassword(creds.Password)
	_, err = env.Db.Query("INSERT INTO " +
		"USERS (email, password_digest, display_name) " +
		"VALUES ($1, $2, $3)",
		creds.Email, passwordDigest, creds.DisplayName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Server Error.")
		return
	}
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	session, _ := env.Store.Get(r, env.Conf.SessionName)
	creds := &structs.UserCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("Attempted login with email: %s", creds.Email)

	var passwordDigest string
	user := structs.User{}
	err = env.Db.QueryRow("select id, image_url, display_name, password_digest from users where email=$1",
		creds.Email).Scan(&user.ID, &user.ImageURL, &user.DisplayName, &passwordDigest)
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
		// Return User info
		b, err := json.Marshal(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write(b)
		}
	default:
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Login Failed as db errored. %v", err)
	}
}

// Facebook Auth
func loginFacebookUser(w http.ResponseWriter, r *http.Request) {
	// checks userId, userToken from FBLoginKit,
	// and returns {"to_register": true} if valid but not registered
	// or user object if valid and registered.
	session, _ := env.Store.Get(r, env.Conf.SessionName)
	creds := &structs.FacebookCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	appToken, err := getFacebookAppToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	isValid := validateFacebookUserToken(appToken, creds.UserToken, creds.UserID)
	if !isValid {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := getUserByEmail(creds.Email)
	if err != nil {
		w.Write([]byte(`{"to_register": true}`))
	} else {
		b, err := json.Marshal(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			session.Values["authenticated"] = true
			session.Save(r, w)
			w.Write(b)
		}
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
	defer resp.Body.Close()
	jsonResp := utils.ReadResponseToJson(resp)
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
	defer resp.Body.Close()
	jsonResp := utils.ReadResponseToJson(resp)["data"].(utils.UnstructuredJSON)
	isValid := jsonResp["is_valid"].(bool) && jsonResp["user_id"].(string) == userId
	return isValid
}

func getUserByEmail(email string) (user structs.User, err error){
	err = env.Db.QueryRow("SELECT id, image_url, display_name FROM users WHERE email=$1",
		email).Scan(&user.ID, &user.ImageURL, &user.DisplayName)
	if err != nil {
		return user, fmt.Errorf("no user found")
	} else {
		return user, nil
	}
}

func registerBySocialMedia(w http.ResponseWriter, r *http.Request) {
	//(email string, displayName string)
	creds := &structs.UserCredentialSocialMedia{}
	err := json.NewDecoder(r.Body).Decode(creds)

	var id string;
	err = env.Db.QueryRow("INSERT INTO USERS (email, display_name, image_url, verified_at) " +
		"VALUES ($1, $2, $3, $4) RETURNING id;",
		creds.Email, creds.DisplayName, creds.ImageUrl, time.Now()).Scan(&id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Server Error.")
		return
	}
	w.Write([]byte(fmt.Sprintf(`{"user_id": "%s"}`, id)))
}