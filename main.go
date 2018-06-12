package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"pebble-dev/rebblestore-api/auth"
	"pebble-dev/rebblestore-api/common"
	"pebble-dev/rebblestore-api/db"
	"pebble-dev/rebblestore-api/rebbleHandlers"

	"github.com/gorilla/handlers"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pborman/getopt"
)

type config struct {
	StoreUrl string `json:"storeUrl"`
	AuthUrl  string `json:"authUrl"`
	HTTPS    bool   `json:"https"`
	Database string `json:"database"`
}

func main() {
	config := config{
		HTTPS:    true,
		StoreUrl: "http://localhost:8081",
		AuthUrl:  "https://localhost:8082",
		Database: "./RebbleAppStore.db",
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
	getopt.StringVarLong(&config.StoreUrl, "store-url", 's', "Set the store URL (defaults to http://localhost:8081)")
	getopt.StringVarLong(&config.AuthUrl, "auth-url", 'a', "Set the auth URL (defaults to https://localhost:8082)")
	getopt.BoolVarLong(&config.HTTPS, "https", 'h', "Set whether or not to use HTTPS (defaults to true)")
	getopt.StringVarLong(&config.Database, "database", 'd', "Specify a specific SQLite database path (defaults to ./RebbleAppStore.db)")
	getopt.Parse()
	if version {
		//fmt.Fprintf(os.Stderr, "Version %s\nBuild Host: %s\nBuild Date: %s\nBuild Hash: %s\n", rsapi.Buildversionstring, rsapi.Buildhost, rsapi.Buildstamp, rsapi.Buildgithash)
		fmt.Fprintf(os.Stderr, "Version %s\nBuild Host: %s\nBuild Date: %s\nBuild Hash: %s\n", common.Buildversionstring, common.Buildhost, common.Buildstamp, common.Buildgithash)
		return
	}

	rebbleHandlers.StoreUrl = config.StoreUrl

	database, err := sql.Open("sqlite3", config.Database)
	if err != nil {
		panic("Could not connect to database" + err.Error())
	}

	dbHandler := db.Handler{database}

	// construct the context that will be injected in to handlers
	context := &rebbleHandlers.HandlerContext{&dbHandler, auth.AuthService{config.AuthUrl}}

	r := rebbleHandlers.Handlers(context)
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	http.Handle("/", r)
	if config.HTTPS {
		err = http.ListenAndServeTLS(":8080", "server.crt", "server.key", loggedRouter)
	} else {
		err = http.ListenAndServe(":8080", loggedRouter)
	}
	if err != nil {
		panic("Could not listen and serve TLS: " + err.Error())
	}
}
