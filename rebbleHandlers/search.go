package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"pebble-dev/rebblestore-api/models"
	"strings"

	"github.com/gorilla/mux"
)

// SearchHandler is the search page
func SearchHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	db := ctx.Database

	if _, ok := mux.Vars(r)["query"]; !ok {
		return http.StatusBadRequest, errors.New("Invalid parameter 'query'")
	}

	query := mux.Vars(r)["query"]
	query = strings.Replace(query, "!", "!!", -1)
	query = strings.Replace(query, "%", "!%", -1)
	query = strings.Replace(query, "_", "!_", -1)
	query = strings.Replace(query, "[", "![", -1)
	query = "%" + query + "%"
	rows, err := db.Query("SELECT id, name, type, thumbs_up, icon_url FROM apps WHERE name LIKE ? ESCAPE '!' ORDER BY thumbs_up DESC LIMIT 12", query)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var cards models.RebbleCards
	cards.Cards = make([]models.RebbleCard, 0)

	for rows.Next() {
		card := models.RebbleCard{}
		err = rows.Scan(&card.Id, &card.Title, &card.Type, &card.ThumbsUp, &card.ImageUrl)
		cards.Cards = append(cards.Cards, card)
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
