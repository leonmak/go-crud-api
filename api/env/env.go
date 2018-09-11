package env

import (
	"groupbuying.online/config"
	"database/sql"
	"github.com/gorilla/sessions"
)

var Conf *config.Configuration
var Db *sql.DB
var Store *sessions.CookieStore
