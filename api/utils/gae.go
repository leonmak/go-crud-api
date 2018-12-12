package utils

import (
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"net/http"
)

func GetAppEngine(r *http.Request, endpoint string) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)
	if appengine.IsAppEngine() {
		ctx := appengine.NewContext(r)
		client := urlfetch.Client(ctx)
		resp, err = client.Get(endpoint)
	} else {
		resp, err = http.Get(endpoint)
	}
	return resp, err
}
