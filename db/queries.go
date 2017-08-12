package db

import (
	"database/sql"
	"encoding/json"
	"errors"
)

// Search returns search results for applications
func Search(handler *sql.DB, query string) (RebbleCards, error) {
	var cards RebbleCards
	rows, err := handler.Query(
		"SELECT id, name, type, thumbs_up, icon_url FROM apps WHERE name LIKE ? ESCAPE '!' ORDER BY thumbs_up DESC LIMIT 12",
		query,
	)
	if err != nil {
		return cards, err
	}
	cards.Cards = make([]RebbleCard, 0)
	for rows.Next() {
		card := RebbleCard{}
		err = rows.Scan(&card.Id, &card.Title, &card.Type, &card.ThumbsUp, &card.ImageUrl)
		cards.Cards = append(cards.Cards, card)
	}
	return cards, nil
}

// GetAppsIDsForCollection returns list of apps for single collection
func GetAppsIDsForCollection(handler *sql.DB, collectionID string) ([]string, error) {
	rows, err := handler.Query("SELECT apps FROM collections WHERE id=?", collectionID)
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return nil, errors.New("Specified collection does not exist")
	}
	var appIdsB []byte
	var appIds []string
	err = rows.Scan(&appIdsB)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(appIdsB, &appIds)
	return appIds, nil
}
