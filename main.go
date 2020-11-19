package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const API_KEY = "cc4d27a67ac610215a1809f87b2cb41b3464f32603b069730f1656947afdb1ea"
const API_URL = "https://min-api.cryptocompare.com/data/price?fsym=BTC&tsyms=USD,JPY,EUR"
const STORE_DATA = "data.json"
const STORE_LAST = "last.json"
const SERVER_PORT = "1234"
const TIME_LOOP_COLLECT_DATA = 10

type CryptoType struct {
	USD  interface{}
	JPY  interface{}
	EUR  interface{}
	Time interface{}
}

func getRequestAPICrypto() (CryptoType, error) {

	log.Info("Collecting data")

	var dataCrypto CryptoType
	req, err := http.NewRequest("GET", API_URL, nil)
	if err != nil {
		return dataCrypto, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Apikey %s", API_KEY))

	client := &http.Client{}
	resp, _ := client.Do(req)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return dataCrypto, err
	}

	json.Unmarshal(body, &dataCrypto)
	dataCrypto.Time = time.Now()

	js, err := json.Marshal(dataCrypto)
	if err != nil {
		return dataCrypto, err
	}

	err = saveFile(STORE_LAST, js)
	if err != nil {
		return dataCrypto, err
	}

	conAll, _ := getFileContent(STORE_DATA)
	var tmpData []CryptoType
	if len(conAll) > 0 {
		err = json.Unmarshal(conAll, &tmpData)
		if err != nil {
			return dataCrypto, err
		}
	}
	tmpData = append(tmpData, dataCrypto)

	jsAll, err := json.Marshal(tmpData)
	if err != nil {
		return dataCrypto, err
	}
	err = saveFile(STORE_DATA, jsAll)
	if err != nil {
		return dataCrypto, err
	}

	return dataCrypto, nil
}

func saveFile(filename string, dataJson []byte) error {
	err := ioutil.WriteFile(filename, dataJson, 0755)
	if err != nil {
		return err
	}
	return nil
}

func getFileContent(filepath string) ([]byte, error) {
	content, err := ioutil.ReadFile(filepath)
	return content, err
}

func routineCollect() {
	for {
		_, err := getRequestAPICrypto()
		if err != nil {
			log.Fatalln(err)
		}
		time.Sleep(TIME_LOOP_COLLECT_DATA * time.Second)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
}

func main() {

	wg := new(sync.WaitGroup)
	wg.Add(2)

	//Collection Data
	go func() {
		routineCollect()
		wg.Done()
	}()

	//API Service
	go func() {
		httpPort := fmt.Sprintf(":%s", SERVER_PORT)
		r := mux.NewRouter().StrictSlash(true)
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

			conAll, err := getFileContent(STORE_DATA)
			if err != nil {
				//respondJSON(w, http.StatusInternalServerError, err.Error())
			}
			conLast, err := getFileContent(STORE_LAST)
			if err != nil {
				//respondJSON(w, http.StatusInternalServerError, err.Error())
			}
			dataResp := map[string]interface{}{
				"last": string(conLast),
				"all":  string(conAll),
			}
			respondJSON(w, http.StatusOK, dataResp)
		})
		log.Info(fmt.Sprintf("Server HTTP running in PORT [%s]", httpPort))
		log.Fatal(http.ListenAndServe(httpPort, r))
		wg.Done()
	}()

	wg.Wait()
}
