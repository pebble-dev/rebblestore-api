package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// SearchHandler is the search page
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	db, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to DB"))
		log.Println(err)
		return
	}
	defer db.Close()

	if _, ok := mux.Vars(r)["query"]; !ok {
		w.WriteHeader(400)
		w.Write([]byte("Missing query parameter"))
		return
	}

	query := mux.Vars(r)["query"]
	query = strings.Replace(query, "!", "!!", -1)
	query = strings.Replace(query, "%", "!%", -1)
	query = strings.Replace(query, "_", "!_", -1)
	query = strings.Replace(query, "[", "![", -1)
	query = "%" + query + "%"
	rows, err := db.Query("SELECT id, name, type, thumbs_up, icon_url FROM apps WHERE name LIKE ? ESCAPE '!' ORDER BY thumbs_up DESC LIMIT 12", query)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to DB"))
		log.Println(err)
		return
	}

	var cards RebbleCards
	cards.Cards = make([]RebbleCard, 0)

	for rows.Next() {
		card := RebbleCard{}
		err = rows.Scan(&card.Id, &card.Title, &card.Type, &card.ThumbsUp, &card.ImageUrl)
		cards.Cards = append(cards.Cards, card)
	}

	data, err := json.MarshalIndent(cards, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}
