package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Handler contains reference to the database client
type Handler struct {
	*sql.DB
}

// Search returns search results for applications
func (handler Handler) Search(query string) (RebbleCards, error) {
	query = strings.Replace(query, "!", "!!", -1)
	query = strings.Replace(query, "%", "!%", -1)
	query = strings.Replace(query, "_", "!_", -1)
	query = strings.Replace(query, "[", "![", -1)
	query = "%" + query + "%"

	var cards RebbleCards
	rows, err := handler.Query(
		"SELECT id, name, type, thumbs_up, screenshots FROM apps WHERE name LIKE ? ESCAPE '!' ORDER BY thumbs_up DESC LIMIT 12",
		query,
	)
	if err != nil {
		return cards, err
	}
	cards.Cards = make([]RebbleCard, 0)
	for rows.Next() {
		card := RebbleCard{}
		var screenshots_b []byte
		var screenshots []RebbleScreenshotsPlatform
		err = rows.Scan(&card.Id, &card.Title, &card.Type, &card.ThumbsUp, &screenshots_b)
		if err != nil {
			return RebbleCards{}, err
		}
		err = json.Unmarshal(screenshots_b, &screenshots)
		if err != nil {
			return RebbleCards{}, err
		}
		if len(screenshots) != 0 && len(screenshots[0].Screenshots) != 0 {
			card.ImageUrl = screenshots[0].Screenshots[0]
		}
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
		rows, err = handler.Query("SELECT id, name, type, thumbs_up, screenshots, published_date, supported_platforms FROM apps WHERE id=?", id)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			app := RebbleApplication{}
			var t int64
			var supported_platforms_b []byte
			var screenshots_b []byte
			err = rows.Scan(&app.Id, &app.Name, &app.Type, &app.ThumbsUp, &screenshots_b, &t, &supported_platforms_b)
			if err != nil {
				return []RebbleApplication{}, err
			}
			app.Published.Time = time.Unix(0, t)
			err = json.Unmarshal(supported_platforms_b, &app.SupportedPlatforms)
			if err != nil {
				return []RebbleApplication{}, err
			}
			err = json.Unmarshal(screenshots_b, &app.Assets.Screenshots)
			if err != nil {
				return []RebbleApplication{}, err
			}
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
