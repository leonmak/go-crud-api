package env

import (
	"database/sql"
	firebase "firebase.google.com/go"
	"github.com/gorilla/sessions"
	"groupbuying.online/api/structs"
)

var (
	Conf *structs.Config
	Db *sql.DB
	Store *sessions.CookieStore
	Firebase *firebase.App
)