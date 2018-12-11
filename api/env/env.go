package env

import (
	"database/sql"
	firebase "firebase.google.com/go"
	"github.com/gorilla/sessions"
	"groupbuying.online/api/structs"
)

var Conf *structs.Config
var Db *sql.DB
var Store *sessions.CookieStore
var Firebase *firebase.App
