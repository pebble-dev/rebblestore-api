package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

// SearchHandler is the search page
func SearchHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	if _, ok := mux.Vars(r)["query"]; !ok {
		return http.StatusBadRequest, errors.New("Invalid parameter 'query'")
	}

	cards, err := ctx.Database.Search(mux.Vars(r)["query"])
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
