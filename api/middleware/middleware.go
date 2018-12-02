package middleware

import (
	"github.com/gorilla/sessions"
	"groupbuying.online/api/structs"
	"net/http"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

// Decorate the request handler with Middleware
func Use(h http.HandlerFunc, middleware ...Middleware) http.HandlerFunc {
	//  r.HandleFunc("/login", use(LoginUser, rateLimit, csrf))
	for _, m := range middleware {
		h = m(h)
	}
	return h
}

func GetAuthMiddleware(store *sessions.CookieStore, conf *structs.Config) Middleware {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			session, _ := store.Get(r, conf.SessionName)
			if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			h(w, r)
		}
	}
}