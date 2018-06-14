package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"regexp"
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
		"SELECT id, name, type, thumbs_up, screenshots FROM apps WHERE name LIKE $1 ESCAPE '!' ORDER BY thumbs_up DESC LIMIT 12",
		query,
	)
	if err != nil {
		log.Printf("SQL error while searching for `%v`", query)
		return cards, err
	}
	cards.Cards = make([]RebbleCard, 0)
	for rows.Next() {
		card := RebbleCard{}
		var screenshots_b []byte
		var screenshots []RebbleScreenshotsPlatform
		err = rows.Scan(&card.Id, &card.Title, &card.Type, &card.ThumbsUp, &screenshots_b)
		if err != nil {
			log.Printf("SQL error: Could not scan search results: %v", err)
			return RebbleCards{}, err
		}
		err = json.Unmarshal(screenshots_b, &screenshots)
		if err != nil {
			log.Printf("Error: Could not unmarshal `screenshots`: %v", err)
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
func (handler Handler) GetAppsForCollection(collectionID string, sortByPopular bool) ([]RebbleApplication, error) {
	var order string

	if sortByPopular {
		order = "thumbs_up"
	} else {
		order = "published_date"
	}

	row := handler.QueryRow("SELECT apps FROM collections WHERE id=$1", collectionID)
	var appIdsB []byte
	var appIds []string
	err := row.Scan(&appIdsB)
	if err != nil {
		log.Printf("SQL error: Could not scan collection: %v", err)
		return nil, err
	}
	json.Unmarshal(appIdsB, &appIds)

	for i, appId := range appIds {
		appIds[i] = "'" + appId + "'"
	}

	idList := strings.Join(appIds, ", ")

	// There is no feasible way for idList to contain user generated data, and therefore for it to be a SQL injection vector. But just in case, we strip any non-authorized character
	reg, err := regexp.Compile("[^a-zA-Z0-9, ']+")
	if err != nil {
		log.Printf("Error: Could not Compile regex: %v", err)
		return nil, err
	}
	idList = reg.ReplaceAllString(idList, "")

	apps := make([]RebbleApplication, 0)

	// It is not possible to give a list or an order via a prepared statement, so this will have to do. We just sanitized idList, so SQL injection isn't a concern.
	rows, err := handler.Query("SELECT id, name, type, thumbs_up, screenshots, published_date, supported_platforms FROM apps WHERE id IN (" + idList + ") ORDER BY " + order + " DESC")
	if err != nil {
		log.Printf("SQL error: Could not query app from collection: %v", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		app := RebbleApplication{}
		var t int64
		var supported_platforms_b []byte
		var screenshots_b []byte
		err = rows.Scan(&app.Id, &app.Name, &app.Type, &app.ThumbsUp, &screenshots_b, &t, &supported_platforms_b)
		if err != nil {
			log.Printf("SQL error: Could not scan app for collection: %v", err)
			return []RebbleApplication{}, err
		}
		app.Published = time.Unix(0, t)
		err = json.Unmarshal(supported_platforms_b, &app.SupportedPlatforms)
		if err != nil {
			log.Printf("Error: Could not unmarshal `supported_platforms`: %v", err)
			return []RebbleApplication{}, err
		}
		err = json.Unmarshal(screenshots_b, &app.Assets.Screenshots)
		if err != nil {
			log.Printf("Error: Could not unmarshal `screenshots`: %v", err)
			return []RebbleApplication{}, err
		}
		apps = append(apps, app)
	}
	return apps, nil
}

// GetCollectionName returns the name of a collection
func (handler Handler) GetCollectionName(collectionID string) (string, error) {
	rows, err := handler.Query("SELECT name FROM collections WHERE id=$1", collectionID)
	if err != nil {
		log.Printf("SQL error: Could not SELECT collection name: %v", err)
		return "", err
	}
	if !rows.Next() {
		return "", errors.New("Specified collection does not exist")
	}
	var name string
	err = rows.Scan(&name)
	if err != nil {
		log.Printf("SQL error: Could not scan collection name: %v", err)
		return "", err
	}

	return name, nil
}

// GetAllApps returns all available apps
func (handler Handler) GetAllApps(sortby string, ascending bool, offset int, limit int) ([]RebbleApplication, error) {
	order := "DESC"
	if ascending {
		order = "ASC"
	}

	var orderCol string
	switch sortby {
	case "popular":
		orderCol = "apps.thumbs_up"
	case "recent":
		orderCol = "apps.published_date"
	default:
		return nil, errors.New("Invalid sortby parameter")
	}

	// this code looks weird, but ORDER BY does not currently work with prepared statements,
	// that is why it is written this way. it should be completely safe as it doesn't take user input
	rows, err := handler.Query(`
		SELECT apps.name, authors.name, apps.icon_url, apps.id, apps.thumbs_up, apps.published_date
		FROM apps
		JOIN authors ON apps.author_id = authors.id
		ORDER BY `+orderCol+" "+order+`
		LIMIT $1
		OFFSET $1
	`, limit, offset)
	if err != nil {
		log.Printf("SQL error: Could not select all apps: %v", err)
		return nil, err
	}
	apps := make([]RebbleApplication, 0)
	for rows.Next() {
		app := RebbleApplication{}
		err = rows.Scan(&app.Name, &app.Author.Name, &app.Assets.Icon, &app.Id, &app.ThumbsUp, &app.Published)

		apps = append(apps, app)
	}
	return apps, nil
}

// GetApp returns a specific app
func (handler Handler) GetApp(id string) (RebbleApplication, error) {
	row := handler.QueryRow(`
		SELECT apps.id, apps.name, apps.author_id, authors.name, apps.tag_ids, apps.description, apps.thumbs_up, apps.type, apps.supported_platforms, apps.published_date, apps.pbw_url, apps.rebble_ready, apps.updated, apps.version, apps.support_url, apps.author_url, apps.source_url, apps.screenshots, apps.banner_url, apps.icon_url, apps.doomsday_backup FROM apps
			JOIN authors ON apps.author_id = authors.id
			WHERE apps.id=$1
	`, id)

	app := RebbleApplication{}
	var supportedPlatforms_b []byte
	var tagIds_b []byte
	var tagIds []string
	var screenshots_b []byte
	var screenshots *([]RebbleScreenshotsPlatform)
	err := row.Scan(&app.Id, &app.Name, &app.Author.Id, &app.Author.Name, &tagIds_b, &app.Description, &app.ThumbsUp, &app.Type, &supportedPlatforms_b, &app.Published, &app.AppInfo.PbwUrl, &app.AppInfo.RebbleReady, &app.AppInfo.Updated, &app.AppInfo.Version, &app.AppInfo.SupportUrl, &app.AppInfo.AuthorUrl, &app.AppInfo.SourceUrl, &screenshots_b, &app.Assets.Banner, &app.Assets.Icon, &app.DoomsdayBackup)
	if err == sql.ErrNoRows {
		return RebbleApplication{}, errors.New("No application with this ID")
	} else if err != nil {
		log.Printf("SQL error: Could not SELECT app: %v", err)
		return RebbleApplication{}, err
	}

	json.Unmarshal(supportedPlatforms_b, &app.SupportedPlatforms)
	json.Unmarshal(tagIds_b, &tagIds)
	app.AppInfo.Tags = make([]RebbleCollection, len(tagIds))
	json.Unmarshal(screenshots_b, &screenshots)
	app.Assets.Screenshots = screenshots

	for i, tagID := range tagIds {
		row := handler.QueryRow("SELECT id, name, color FROM collections WHERE id=$1", tagID)

		err = row.Scan(&app.AppInfo.Tags[i].Id, &app.AppInfo.Tags[i].Name, &app.AppInfo.Tags[i].Color)
		if err != nil {
			log.Printf("SQL error: Could not scan app: %v", err)
			return RebbleApplication{}, err
		}
	}

	return app, nil
}

// GetAppTags returns the the list of tags of the application with the id `id`
func (handler Handler) GetAppTags(id string) ([]RebbleCollection, error) {
	rows, err := handler.Query("SELECT apps.tag_ids FROM apps WHERE id=$1", id)
	if err != nil {
		log.Printf("SQL error: Could not SELECT app tags: %v", err)
		return []RebbleCollection{}, err
	}
	exists := rows.Next()
	if !exists {
		log.Printf("Error: Tag `%v` doesn't exist!", id)
		return []RebbleCollection{}, err
	}

	var tagIds_b []byte
	var tagIds []string
	err = rows.Scan(&tagIds_b)
	if err != nil {
		log.Printf("SQL error: Could not scan app tags: %v", err)
		return []RebbleCollection{}, err
	}
	json.Unmarshal(tagIds_b, &tagIds)
	collections := make([]RebbleCollection, len(tagIds))

	for i, tagId := range tagIds {
		rows, err := handler.Query("SELECT id, name, color FROM collections WHERE id=$1", tagId)
		if err != nil {
			log.Printf("SQL error: Could not SELECT collection from app tag: %v", err)
			return []RebbleCollection{}, err
		}

		rows.Next()
		err = rows.Scan(&collections[i].Id, &collections[i].Name, &collections[i].Color)
		if err != nil {
			log.Printf("SQL error: Could not scan collection from app tag: %v", err)
			return []RebbleCollection{}, err
		}
	}

	return collections, nil
}

// GetAppVersions returns the the list of versions of the application with the id `id`
func (handler Handler) GetAppVersions(id string) ([]RebbleVersion, error) {
	rows, err := handler.Query("SELECT apps.versions FROM apps WHERE id=$1", id)
	if err != nil {
		log.Printf("SQL error: Could not SELECT app versions: %v", err)
		return []RebbleVersion{}, err
	}
	exists := rows.Next()
	if !exists {
		return []RebbleVersion{}, errors.New("No app with this ID")
	}

	var versions_b []byte
	var versions []RebbleVersion
	err = rows.Scan(&versions_b)
	if err != nil {
		log.Printf("SQL error: Could not scan app versions: %v", err)
		return []RebbleVersion{}, err
	}
	json.Unmarshal(versions_b, &versions)

	return versions, nil
}

// GetAuthor returns a RebbleAuthor
func (handler Handler) GetAuthor(id int) (RebbleAuthor, error) {
	rows, err := handler.Query("SELECT authors.name FROM authors WHERE id=$1", id)
	if err != nil {
		log.Printf("SQL error: Could not SELECT author: %v", err)
		return RebbleAuthor{}, err
	}
	exists := rows.Next()
	if !exists {
		return RebbleAuthor{}, errors.New("No app with this ID")
	}

	author := RebbleAuthor{
		Id: id,
	}
	err = rows.Scan(&author.Name)
	if err != nil {
		log.Printf("SQL error: Could not scan author: %v", err)
		return RebbleAuthor{}, err
	}

	return author, nil
}

// GetAuthorCards returns cards for all apps from a specific author
func (handler Handler) GetAuthorCards(id int) (RebbleCards, error) {
	rows, err := handler.Query(`
		SELECT id, name, type, screenshots, thumbs_up
		FROM apps
		WHERE author_id=$1
		ORDER BY published_date ASC
	`, id)
	if err != nil {
		log.Printf("SQL error: Could not SELECT author cards: %v", err)
		return RebbleCards{}, err
	}

	cards := RebbleCards{
		Cards: make([]RebbleCard, 0),
	}

	for rows.Next() {
		card := RebbleCard{}

		var screenshots_b []byte
		var screenshots []RebbleScreenshotsPlatform

		err = rows.Scan(&card.Id, &card.Title, &card.Type, &screenshots_b, &card.ThumbsUp)
		if err != nil {
			log.Printf("SQL error: Could not scan author card: %v", err)
			return RebbleCards{}, err
		}

		err = json.Unmarshal(screenshots_b, &screenshots)
		if err != nil {
			log.Printf("Error: could not unmarshal author card `screenshots`", err)
			return RebbleCards{}, err
		}
		if len(screenshots) != 0 && len(screenshots[0].Screenshots) != 0 {
			card.ImageUrl = screenshots[0].Screenshots[0]
		}
		cards.Cards = append(cards.Cards, card)
	}

	return cards, nil
}
