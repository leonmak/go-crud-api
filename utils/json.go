package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
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

func WriteSuccessJsonResponse(w http.ResponseWriter, message string) {
	WriteJsonResponse(w, "success", message)
}

func WriteErrorJsonResponse(w http.ResponseWriter, message string) {
	WriteJsonResponse(w, "error", message)
}

func WriteJsonResponse(w http.ResponseWriter, key string, values ...interface{}) {
	if len(values) == 1 {
		jsonRespMap := make(map[string]interface{})
		jsonRespMap[key] = values[0]
		jsonResp, err := json.Marshal(jsonRespMap)
		if err == nil {
			w.Write(jsonResp)
		} else {
			WriteErrorJsonResponse(w, err.Error())
		}
	} else {
		jsonRespMap := make(map[string][]interface{})
		jsonRespMap[key] = values
		jsonResp, err := json.Marshal(jsonRespMap)
		if err == nil {
			w.Write(jsonResp)
		} else {
			WriteErrorJsonResponse(w, err.Error())
		}
	}
}
