package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func insert(cards *([]RebbleCard), location int, card RebbleCard) *([]RebbleCard) {
	beggining := (*cards)[:location]
	end := make([]RebbleCard, len(*cards)-len(beggining))
	copy(end, (*cards)[location:])
	beggining = append(beggining, card)
	beggining = append(beggining, end...)

	return &beggining
}

func remove(cards *([]RebbleCard), location int) *([]RebbleCard) {
	new := make([]RebbleCard, location)
	copy(new, (*cards)[:location])
	new = append(new, (*cards)[location+1:]...)

	return &new
}

func bestCards(cards *([]RebbleCard), sortByPopular bool, nCards int) *([]RebbleCard) {
	newCards := make([]RebbleCard, nCards)
	copy(newCards, *cards)

	for _, card := range *cards {
		newCards = append(newCards, card)

		if len(newCards) > nCards {
			if sortByPopular {
				worst := 0
				for i, newCard := range newCards {
					if newCard.ThumbsUp < newCards[worst].ThumbsUp {
						worst = i
					}
				}
				newCards = *(remove(&newCards, worst))
			} else {
				worst := 0
				for i, newCard := range newCards {
					if newCard.Published.UnixNano() < newCards[worst].Published.UnixNano() {
						worst = i
					}
				}
				newCards = *(remove(&newCards, worst))
			}
		}
	}

	return &newCards
}

func sortCards(cards *([]RebbleCard), sortByPopular bool) *([]RebbleCard) {
	newCards := make([]RebbleCard, 0)

	for _, card := range *cards {
		if len(newCards) == 0 {
			newCards = []RebbleCard{card}

			continue
		} else if len(newCards) == 1 {
			if sortByPopular {
				if newCards[0].ThumbsUp > card.ThumbsUp {
					newCards = []RebbleCard{newCards[0], card}
				} else {
					newCards = []RebbleCard{card, newCards[0]}
				}
			} else {
				if newCards[0].Published.UnixNano() > card.Published.UnixNano() {
					newCards = []RebbleCard{card, newCards[0]}
				} else {
					newCards = []RebbleCard{newCards[0], card}
				}
			}

			continue
		}

		if sortByPopular {
			for i, newCard := range newCards {
				if newCard.ThumbsUp < card.ThumbsUp {
					newCards = *(insert(&newCards, i, card))
					break
				}
			}
		} else {
			for i, newCard := range newCards {
				if card.Published.UnixNano() > newCard.Published.UnixNano() {
					newCards = *(insert(&newCards, i, card))
					break
				}
			}
		}
	}

	return &newCards
}

// CollectionHandler serves a list of cards from a collection
func CollectionHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	db, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to DB"))
		log.Println(err)
		return
	}
	defer db.Close()

	urlquery := r.URL.Query()

	if _, ok := mux.Vars(r)["id"]; !ok {
		w.WriteHeader(400)
		w.Write([]byte("Missing id parameter"))
		return
	}

	var sortByPopular bool
	if o, ok := urlquery["order"]; ok {
		if len(o) > 1 {
			w.WriteHeader(400)
			w.Write([]byte("Multiple order types are not allowed"))
			return
		} else if o[0] == "popular" {
			sortByPopular = true
		} else if o[0] == "new" {
			sortByPopular = false
		} else {
			w.WriteHeader(400)
			w.Write([]byte("Invalid order parameter"))
			return
		}
	}

	rows, err := db.Query("SELECT apps FROM collections WHERE id=?", mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to DB"))
		log.Println(err)
		return
	}
	if !rows.Next() {
		w.WriteHeader(500)
		w.Write([]byte("Specified collection does not exist"))
		return
	}
	var appIds_b []byte
	var appIds []string
	err = rows.Scan(&appIds_b)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(appIds_b, &appIds)

	var cards RebbleCards
	cards.Cards = make([]RebbleCard, 0)
	for _, id := range appIds {
		rows, err = db.Query("SELECT id, name, type, thumbs_up, icon_url, published_date FROM apps WHERE id=?", id)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Unable to connect to DB"))
			log.Println(err)
			return
		}

		for rows.Next() {
			card := RebbleCard{}
			var t int64
			err = rows.Scan(&card.Id, &card.Title, &card.Type, &card.ThumbsUp, &card.ImageUrl, &t)
			card.Published.Time = time.Unix(0, t)
			cards.Cards = append(cards.Cards, card)
		}
	}

	cards.Cards = *(bestCards(&cards.Cards, sortByPopular, 12))
	cards.Cards = *(sortCards(&cards.Cards, sortByPopular))

	data, err := json.MarshalIndent(cards, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}
