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
	err = env.Db.QueryRow("select password_digest from users where email=$1",
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

func loginFacebookUser(w http.ResponseWriter, r *http.Request) {
	session, _ := env.Store.Get(r, env.Conf.SessionName)
	creds := &structs.UserCredentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

}