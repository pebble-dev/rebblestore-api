package db

import (
	"database/sql"
	"encoding/json"
	"errors"
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
func (handler Handler) GetAppsForCollection(collectionID string, sortByPopular bool) ([]RebbleApplication, error) {
	var order string

	if sortByPopular {
		order = "thumbs_up"
	} else {
		order = "published_date"
	}

	row := handler.QueryRow("SELECT apps FROM collections WHERE id=?", collectionID)
	var appIdsB []byte
	var appIds []string
	err := row.Scan(&appIdsB)
	if err != nil {
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
		return nil, err
	}
	idList = reg.ReplaceAllString(idList, "")

	apps := make([]RebbleApplication, 0)

	// It is not possible to give a list or an order via a prepared statement, so this will have to do. We just sanitized idList, so SQL injection isn't a concern.
	rows, err := handler.Query("SELECT id, name, type, thumbs_up, screenshots, published_date, supported_platforms FROM apps WHERE id IN (" + idList + ") ORDER BY " + order + " DESC")
	if err != nil {
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
	return apps, nil
}

// GetCollectionName returns the name of a collection
func (handler Handler) GetCollectionName(collectionID string) (string, error) {
	rows, err := handler.Query("SELECT name FROM collections WHERE id=?", collectionID)
	if err != nil {
		return "", err
	}
	if !rows.Next() {
		return "", errors.New("Specified collection does not exist")
	}
	var name string
	err = rows.Scan(&name)
	if err != nil {
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
		LIMIT ?
		OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	apps := make([]RebbleApplication, 0)
	for rows.Next() {
		app := RebbleApplication{}
		var t_published int64
		err = rows.Scan(&app.Name, &app.Author.Name, &app.Assets.Icon, &app.Id, &app.ThumbsUp, &t_published)
		app.Published.Time = time.Unix(0, t_published)

		apps = append(apps, app)
	}
	return apps, nil
}

// GetApp returns a specific app
func (handler Handler) GetApp(id string) (RebbleApplication, error) {
	row := handler.QueryRow("SELECT apps.id, apps.name, apps.author_id, authors.name, apps.tag_ids, apps.description, apps.thumbs_up, apps.type, apps.supported_platforms, apps.published_date, apps.pbw_url, apps.rebble_ready, apps.updated, apps.version, apps.support_url, apps.author_url, apps.source_url, apps.screenshots, apps.banner_url, apps.icon_url, apps.doomsday_backup FROM apps JOIN authors ON apps.author_id = authors.id WHERE apps.id=?", id)

	app := RebbleApplication{}
	var supportedPlatforms_b []byte
	var t_published, t_updated int64
	var tagIds_b []byte
	var tagIds []string
	var screenshots_b []byte
	var screenshots *([]RebbleScreenshotsPlatform)
	err := row.Scan(&app.Id, &app.Name, &app.Author.Id, &app.Author.Name, &tagIds_b, &app.Description, &app.ThumbsUp, &app.Type, &supportedPlatforms_b, &t_published, &app.AppInfo.PbwUrl, &app.AppInfo.RebbleReady, &t_updated, &app.AppInfo.Version, &app.AppInfo.SupportUrl, &app.AppInfo.AuthorUrl, &app.AppInfo.SourceUrl, &screenshots_b, &app.Assets.Banner, &app.Assets.Icon, &app.DoomsdayBackup)
	if err == sql.ErrNoRows {
		return RebbleApplication{}, errors.New("No application with this ID")
	} else if err != nil {
		return RebbleApplication{}, err
	}

	json.Unmarshal(supportedPlatforms_b, &app.SupportedPlatforms)
	app.Published.Time = time.Unix(0, t_published)
	app.AppInfo.Updated.Time = time.Unix(0, t_updated)
	json.Unmarshal(tagIds_b, &tagIds)
	app.AppInfo.Tags = make([]RebbleCollection, len(tagIds))
	json.Unmarshal(screenshots_b, &screenshots)
	app.Assets.Screenshots = screenshots

	for i, tagID := range tagIds {
		row := handler.QueryRow("SELECT id, name, color FROM collections WHERE id=?", tagID)

		err = row.Scan(&app.AppInfo.Tags[i].Id, &app.AppInfo.Tags[i].Name, &app.AppInfo.Tags[i].Color)
		if err != nil {
			return RebbleApplication{}, err
		}
	}

	return app, nil
}

// GetAppTags returns the the list of tags of the application with the id `id`
func (handler Handler) GetAppTags(id string) ([]RebbleCollection, error) {
	rows, err := handler.Query("SELECT apps.tag_ids FROM apps WHERE id=?", id)
	if err != nil {
		return []RebbleCollection{}, err
	}
	exists := rows.Next()
	if !exists {
		return []RebbleCollection{}, err
	}

	var tagIds_b []byte
	var tagIds []string
	err = rows.Scan(&tagIds_b)
	if err != nil {
		return []RebbleCollection{}, err
	}
	json.Unmarshal(tagIds_b, &tagIds)
	collections := make([]RebbleCollection, len(tagIds))

	for i, tagId := range tagIds {
		rows, err := handler.Query("SELECT id, name, color FROM collections WHERE id=?", tagId)
		if err != nil {
			return []RebbleCollection{}, err
		}

		rows.Next()
		err = rows.Scan(&collections[i].Id, &collections[i].Name, &collections[i].Color)
		if err != nil {
			return []RebbleCollection{}, err
		}
	}

	return collections, nil
}

// GetAppVersions returns the the list of versions of the application with the id `id`
func (handler Handler) GetAppVersions(id string) ([]RebbleVersion, error) {
	rows, err := handler.Query("SELECT apps.versions FROM apps WHERE id=?", id)
	if err != nil {
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
		return []RebbleVersion{}, err
	}
	json.Unmarshal(versions_b, &versions)

	return versions, nil
}

// GetAuthor returns a RebbleAuthor
func (handler Handler) GetAuthor(id int) (RebbleAuthor, error) {
	rows, err := handler.Query("SELECT authors.name FROM authors WHERE id=?", id)
	if err != nil {
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
		return RebbleAuthor{}, err
	}

	return author, nil
}

// GetAuthorCards returns cards for all apps from a specific author
func (handler Handler) GetAuthorCards(id int) (RebbleCards, error) {
	rows, err := handler.Query(`
		SELECT id, name, type, screenshots, thumbs_up
		FROM apps
		WHERE author_id=?
		ORDER BY published_date ASC
	`, id)
	if err != nil {
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
