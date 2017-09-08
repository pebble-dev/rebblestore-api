package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"pebble-dev/rebblestore-api/common"
	"pebble-dev/rebblestore-api/db"
	"pebble-dev/rebblestore-api/rebbleHandlers"

	"github.com/gorilla/handlers"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pborman/getopt"
)

type config struct {
	CaptchaSecret string `json:"captchaSecret"`
	StoreUrl      string `json:"storeUrl"`
}

func main() {
	config := config{
		StoreUrl: "https://docs.rebble.io",
	}

	file, err := ioutil.ReadFile("./rebblestore-api.json")
	if err != nil {
		panic("Could not load rebblestore-api.json: " + err.Error())
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		panic("Could not parse rebblestore-api.json: " + err.Error())
	}

	var version bool

	getopt.BoolVarLong(&version, "version", 'V', "Get the current version info")
	getopt.StringVarLong(&config.StoreUrl, "store-url", 'u', "Set the store URL (defaults to http://docs.rebble.io)")
	getopt.StringVarLong(&config.CaptchaSecret, "captcha-secret", 'c', "Set the ReCAPTCHA secret key (registration won't work without it)")
	getopt.Parse()
	if version {
		//fmt.Fprintf(os.Stderr, "Version %s\nBuild Host: %s\nBuild Date: %s\nBuild Hash: %s\n", rsapi.Buildversionstring, rsapi.Buildhost, rsapi.Buildstamp, rsapi.Buildgithash)
		fmt.Fprintf(os.Stderr, "Version %s\nBuild Host: %s\nBuild Date: %s\nBuild Hash: %s\n", common.Buildversionstring, common.Buildhost, common.Buildstamp, common.Buildgithash)
		return
	}

	rebbleHandlers.StoreUrl = config.StoreUrl

	database, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		panic("Could not connect to database" + err.Error())
	}

	dbHandler := db.Handler{database}

	// construct the context that will be injected in to handlers
	context := &rebbleHandlers.HandlerContext{&dbHandler, config.CaptchaSecret}

	r := rebbleHandlers.Handlers(context)
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	http.Handle("/", r)
	err = http.ListenAndServeTLS(":8080", "server.crt", "server.key", loggedRouter)
	if err != nil {
		panic("Could not listen and serve TLS: " + err.Error())
	}
}
