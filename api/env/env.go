package env

import (
	"database/sql"
	"github.com/gorilla/sessions"
	"groupbuying.online/api/structs"
)

var Conf *structs.Config
var Db *sql.DB
var Store *sessions.CookieStore
