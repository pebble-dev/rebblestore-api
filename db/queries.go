package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Handler contains reference to the database client
type Handler struct {
	*sql.DB
}

// Search returns search results for applications
func (handler Handler) Search(query string) (RebbleCards, error) {
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
func (handler Handler) GetAppsForCollection(collectionID string) ([]RebbleApplication, error) {
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

// GetApps returns all available apps
func (handler Handler) GetApps() ([]RebbleApplication, error) {
	rows, err := handler.Query(`
		SELECT apps.name, authors.name
		FROM apps
		JOIN authors ON apps.author_id = authors.id
		ORDER BY published_date ASC LIMIT 20
	`)
	if err != nil {
		return nil, err
	}
	apps := make([]RebbleApplication, 0)
	for rows.Next() {
		app := RebbleApplication{}
		err = rows.Scan(&app.Name, &app.Author.Name)
		apps = append(apps, app)
	}
	return apps, nil
}

// GetApp
// func GetApp(handler *sql.DB, id string) (RebbleApplication, error) {
// 	return nil, nil
// }
