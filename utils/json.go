package utils

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type UnstructuredJSON = map[string]interface{}

func ReadRequestToJson(r *http.Request) (UnstructuredJSON, error) {
	var result UnstructuredJSON
	jsonRead, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(jsonRead), &result)
	return result, nil
}

func ReadResponseToJson(resp *http.Response) (respJson UnstructuredJSON) {
	json.NewDecoder(resp.Body).Decode(&respJson)
	return respJson
}
