package routes

import (
	"github.com/gorilla/mux"
	"groupbuying.online/api/middleware"
	"groupbuying.online/api/env"
	"net/http"
	"fmt"
	"log"
)

func InitRouter() {
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	auth := middleware.GetAuthMiddleware(env.Store, env.Conf)

	// Deal
	api.HandleFunc("/deals", getDeals).Methods(http.MethodGet)
	api.HandleFunc("/deals", middleware.Use(postDeal, auth)).Methods(http.MethodPost)
	api.HandleFunc("/deal/{dealId}",
		middleware.Use(handleDeal, auth)).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/memberships", getDealMembersByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/membership/{userId}",
		middleware.Use(handleDealMembership, auth)).Methods(http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/likes", getDealLikeSummaryByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/like/{userId}",
		middleware.Use(handleDealLike, auth)).Methods(http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/images", getDealImageUrlsByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/image/{userId}",
		middleware.Use(handleDealImage, auth)).Methods(http.MethodPost, http.MethodDelete)

	api.HandleFunc("/deal/{dealId}/comments", getDealCommentsByDealId).Methods(http.MethodGet)
	api.HandleFunc("/deal/{dealId}/comment/{userId}",
		middleware.Use(handleDealComment, auth)).Methods(http.MethodPost, http.MethodDelete)

	// User
	// TODO: Get another user's profile stats
	api.HandleFunc("/register", createUser).Methods(http.MethodPost)
	api.HandleFunc("/register_social_media", registerBySocialMedia).Methods(http.MethodPost)
	api.HandleFunc("/login", loginUser).Methods(http.MethodPost)
	api.HandleFunc("/login_facebook", loginFacebookUser).Methods(http.MethodPost)
	api.HandleFunc("/logout", middleware.Use(logoutUser, auth)).Methods(http.MethodPost)

	fmt.Printf("listening on %d\n", env.Conf.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", env.Conf.Port), api))
}
