package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"pebble-dev/rebblestore-api/common"
	"pebble-dev/rebblestore-api/db"
	"pebble-dev/rebblestore-api/rebbleHandlers"

	"github.com/gorilla/handlers"
	_ "github.com/lib/pq"
	"github.com/pborman/getopt"
)

func main() {
	var version bool
	rebbleHandlers.StoreUrl = "http://docs.rebble.io"
	getopt.BoolVarLong(&version, "version", 'V', "Get the current version info")
	getopt.StringVarLong(&rebbleHandlers.StoreUrl, "store-url", 'u', "Set the store URL (defaults to http://docs.rebble.io)")
	getopt.Parse()
	if version {
		//fmt.Fprintf(os.Stderr, "Version %s\nBuild Host: %s\nBuild Date: %s\nBuild Hash: %s\n", rsapi.Buildversionstring, rsapi.Buildhost, rsapi.Buildstamp, rsapi.Buildgithash)
		fmt.Fprintf(os.Stderr, "Version %s\nBuild Host: %s\nBuild Date: %s\nBuild Hash: %s\n", common.Buildversionstring, common.Buildhost, common.Buildstamp, common.Buildgithash)
		return
	}

	database, err := sql.Open("postgres", "user=rebblestore dbname=rebblestore")
	if err != nil {
		panic("Could not connect to database" + err.Error())
	}

	dbHandler := db.Handler{database}

	// construct the context that will be injected in to handlers
	context := &rebbleHandlers.HandlerContext{&dbHandler}

	r := rebbleHandlers.Handlers(context)
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	http.Handle("/", r)
	http.ListenAndServe(":8080", loggedRouter)
}
