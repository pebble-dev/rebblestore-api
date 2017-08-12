package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
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

// GetAppsForCollection returns list of apps for single collection
func GetAppsForCollection(handler *sql.DB, collectionID string) ([]RebbleApplication, error) {
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

	apps := make([]RebbleApplication, 0)
	for _, id := range appIds {
		rows, err = handler.Query("SELECT id, name, type, thumbs_up, icon_url, published_date, supported_platforms FROM apps WHERE id=?", id)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			app := RebbleApplication{}
			var t int64
			var supported_platforms_b []byte
			err = rows.Scan(&app.Id, &app.Name, &app.Type, &app.ThumbsUp, &app.Assets.Icon, &t, &supported_platforms_b)
			app.Published.Time = time.Unix(0, t)
			err = json.Unmarshal(supported_platforms_b, &app.SupportedPlatforms)
			apps = append(apps, app)
		}
	}
	return apps, nil
}
