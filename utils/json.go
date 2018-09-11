package utils

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type UnstructuredJSON = map[string]interface{}

func ReadUnstructuredJson(r *http.Request) (UnstructuredJSON, error) {
	var result UnstructuredJSON
	jsonRead, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(jsonRead), &result)
	return result, nil
}
