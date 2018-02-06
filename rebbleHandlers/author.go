package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"pebble-dev/rebblestore-api/db"
	"strconv"

	"github.com/gorilla/mux"
)

type rebbleAuthor struct {
	Id    int             `json:"id"`
	Name  string          `json:"name"`
	Cards []db.RebbleCard `json:"cards"`
}

func AuthorHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return http.StatusBadRequest, errors.New("Non-numeric ID")
	}

	author, err := ctx.Database.GetAuthor(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	cards, err := ctx.Database.GetAuthorCards(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	result := rebbleAuthor{
		Id:    author.Id,
		Name:  author.Name,
		Cards: cards.Cards,
	}

	data, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)

	return http.StatusOK, nil
}
