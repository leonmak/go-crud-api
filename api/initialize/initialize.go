package initialize

import (
	"log"
	"net/http"
	"database/sql"
	"os"
	"encoding/json"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"

	"groupbuying.online/config"
	"groupbuying.online/api/routes"
	"groupbuying.online/api/middleware"
	"groupbuying.online/api/env"
)

// TODO: Implement:
// TODO: - csrf, rate limit middleware
// TODO: - cloudinary, email verification, change name/password

func Init() {
	initGlobal()
	initRouter()
}

func initGlobal() {
	initConfig()
	initDB()
	initSessionStore()
}

func initRouter() {
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	auth := middleware.GetAuthMiddleware(env.Store, env.Conf)

	// Deal
	api.HandleFunc("/deals", routes.GetDeals).Methods(http.MethodGet)
	api.HandleFunc("/deals", middleware.Use(routes.PostDeal, auth)).Methods(http.MethodPost)
	api.HandleFunc("/deal/{dealId}", middleware.Use(routes.HandleDeal, auth)).Methods(
		http.MethodGet, http.MethodPut, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/memberships", routes.GetDealMembersByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/membership/{userId}", middleware.Use(routes.HandleDealMembership, auth)).Methods(
		http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/likes", routes.GetDealLikeSummaryByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/like/{userId}", middleware.Use(routes.HandleDealLike, auth)).Methods(
		http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/images", routes.GetDealImageUrlsByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/image/{userId}", middleware.Use(routes.HandleDealImage, auth)).Methods(
		http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/comments", routes.GetDealCommentsByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/comment/{userId}", middleware.Use(routes.HandleDealComment, auth)).Methods(
		http.MethodPost, http.MethodDelete)

	// User
	// TODO: Get another user's profile stats
	api.HandleFunc("/register", routes.CreateUser).Methods(http.MethodPost)
	api.HandleFunc("/login", routes.LoginUser).Methods(http.MethodPost)
	api.HandleFunc("/logout", middleware.Use(routes.LogoutUser, auth)).Methods(http.MethodPost)

	fmt.Printf("listening on %d\n", env.Conf.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", env.Conf.Port), api))
}

func initSessionStore() {
	key := []byte(env.Conf.SessionStoreKey)
	env.Store = sessions.NewCookieStore(key)
}

func initDB() {
	var err error
	connStr := fmt.Sprintf("dbname=%s user=%s password=%s sslmode=disable",
		env.Conf.DBSourceName, env.Conf.DBUsername, env.Conf.DBPassword)
	env.Db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
}

func getConfiguration(envType string) (*config.Configuration, error) {
	if envType == "" {
		envType = "dev"
	}
	var configuration config.Configuration
	file, err := os.Open("config/" + envType + ".json")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	if err != nil {
		return nil, err
	}
	return &configuration, err
}

func initConfig() {
	var err error
	env.Conf, err = getConfiguration(os.Getenv("GO_ENV"))
	if err != nil {
		log.Fatal(err)
	}
}
