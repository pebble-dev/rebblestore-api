package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"pebble-dev/rebblestore-api/db"
	"strings"

	"github.com/gorilla/mux"
)

// SearchHandler is the search page
func SearchHandler(ctx *handlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	if _, ok := mux.Vars(r)["query"]; !ok {
		return http.StatusBadRequest, errors.New("Invalid parameter 'query'")
	}

	query := mux.Vars(r)["query"]
	query = strings.Replace(query, "!", "!!", -1)
	query = strings.Replace(query, "%", "!%", -1)
	query = strings.Replace(query, "_", "!_", -1)
	query = strings.Replace(query, "[", "![", -1)
	query = "%" + query + "%"
	var cards db.RebbleCards
	cards, err := db.Search(ctx.db, query)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	data, err := json.MarshalIndent(cards, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)

	return http.StatusOK, nil
}
