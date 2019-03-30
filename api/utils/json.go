package utils

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type UnstructuredJSON = map[string]interface{}

func ReadRequestToJson(r *http.Request) (result UnstructuredJSON, err error) {
	jsonRead, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(jsonRead), &result)
	return result, err
}

func ReadResponseToJson(resp *http.Response) (respJson UnstructuredJSON, err error) {
	err = json.NewDecoder(resp.Body).Decode(&respJson)
	return respJson, err
}

func WriteSuccessJsonResponse(w http.ResponseWriter, message string) {
	WriteJsonResponse(w, "success", message)
}

func WriteErrorJsonResponse(w http.ResponseWriter, message string) {
	WriteJsonResponse(w, "error", message)
}

// Throws an error after logging and writing to response
func CheckFatalError(w http.ResponseWriter, err error) {
	if err != nil {
		WriteErrorJsonResponse(w, err.Error())
		log.Fatal(err.Error())
	}
}

func WriteJsonResponse(w http.ResponseWriter, key string, values ...interface{}) {
	if len(values) == 1 {
		jsonRespMap := make(map[string]interface{})
		jsonRespMap[key] = values[0]
		jsonResp, err := json.Marshal(jsonRespMap)
		if err == nil {
			_, err = w.Write(jsonResp)
		} else {
			WriteErrorJsonResponse(w, err.Error())
		}
	} else {
		jsonRespMap := make(map[string][]interface{})
		jsonRespMap[key] = values
		jsonResp, err := json.Marshal(jsonRespMap)
		if err == nil {
			_, err = w.Write(jsonResp)
		} else {
			WriteErrorJsonResponse(w, err.Error())
		}
	}
}
