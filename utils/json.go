package utils

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
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

func WriteJsonResponse(w http.ResponseWriter, key string, values ...interface{}) {
	if len(values) == 1 {
		jsonRespMap := make(map[string]interface{})
		jsonRespMap[key] = values[0]
		jsonResp, err := json.Marshal(jsonRespMap)
		if err == nil {
			w.Write(jsonResp)
		} else {
			fmt.Errorf(err.Error())
		}
	} else {
		jsonRespMap := make(map[string][]interface{})
		jsonRespMap[key] = values
		jsonResp, err := json.Marshal(jsonRespMap)
		if err == nil {
			w.Write(jsonResp)
		} else {
			fmt.Errorf(err.Error())
		}
	}
}
