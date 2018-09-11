package utils

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type unstructuredJSON = map[string]interface{}

func ReadUnstructuredJson(r *http.Request) (unstructuredJSON, error) {
	var result unstructuredJSON
	jsonRead, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(jsonRead), &result)
	return result, nil
}
