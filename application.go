package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// PebbleAppList contains a list of PebbleApplication. It matches the format of Pebble API answers.
type PebbleAppList struct {
	Apps []*PebbleApplication `json:"data"`
}

// RebbleApplication contains Pebble App information from the DB
type RebbleApplication struct {
	Id                 string        `json:"id"`
	Name               string        `json:"title"`
	Author             RebbleAuthor  `json:"author"`
	Description        string        `json:"description"`
	ThumbsUp           int           `json:"thumbs_up"`
	Type               string        `json:"type"`
	SupportedPlatforms []string      `json:"supported_platforms"`
	Published          JSONTime      `json:"published_date"`
	AppInfo            RebbleAppInfo `json:"appInfo"`
	Assets             RebbleAssets  `json:"assets"`
	DoomsdayBackup     bool          `json:"doomsday_backup"`
}

// RebbleTagList contains a list of tag. Used by getApi(id)
type RebbleTagList struct {
	Tags []RebbleCollection `json:"tags"`
}

// RebbleChangelog contains a list of version changes for an app
type RebbleChangelog struct {
	Versions []RebbleVersion `json:"versions"`
}

// RebbleVersion contains information about a specific version of an app
type RebbleVersion struct {
	Number      string   `json:"number"`
	ReleaseDate JSONTime `json:"release_date"`
	Description string   `json:"description"`
}

// RebbleAppInfo contains information about the app (pbw url, versioning, links, etc.)
type RebbleAppInfo struct {
	PbwUrl      string             `json:"pbwUrl"`
	RebbleReady bool               `json:"rebbleReady"`
	Tags        []RebbleCollection `json:"tags"`
	Updated     JSONTime           `json:"updated"`
	Version     string             `json:"version"`
	SupportUrl  string             `json:"supportUrl"`
	AuthorUrl   string             `json:"authorUrl"`
	SourceUrl   string             `json:"sourceUrl"`
}

// RebbleAuthor describes the autor of a Rebble app (ID and name)
type RebbleAuthor struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// RebbleAssets describes the list of assets of a Rebble app (banner, icon, screenshots)
type RebbleAssets struct {
	Banner      string                         `json:"appBanner"`
	Icon        string                         `json:"appIcon"`
	Screenshots *([]RebbleScreenshotsPlatform) `json:"screenshots"`
}

// RebbleScreenshotsPlatform contains a list of screenshots specific to some hardware (since each Pebble watch renders UI differently)
type RebbleScreenshotsPlatform struct {
	Platform    string   `json:"platform"`
	Screenshots []string `json:"screenshots"`
}

// RebbleCollection describes the collection (category) of a Rebble application
type RebbleCollection struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// PebbleApplication is used by the parseApp() function. It matches directly the `{id}.json` format.
type PebbleApplication struct {
	Id                 string                   `json:"id"`
	Name               string                   `json:"title"`
	Author             string                   `json:"author"`
	CategoryId         string                   `json:"category_id"`
	CategoryName       string                   `json:"category_name"`
	CategoryColor      string                   `json:"category_color"`
	Description        string                   `json:"description"`
	Published          JSONTime                 `json:"published_date"`
	Release            PebbleApplicationRelease `json:"latest_release"`
	Website            string                   `json:"website"`
	Source             string                   `json:"source"`
	Screenshots        PebbleScreenshotImages   `json:"screenshot_images"`
	ScreenshotHardware string                   `json:"screenshot_hardware"`
	HeaderImages       PebbleHeaderImages       `json:"header_images"`
	Hearts             int                      `json:"hearts"`
	Type               string                   `json:"type"`
	Compatibility      PebbleCompatibility      `json:"compatibility"`
	Changelog          []PebbleVersion          `json:"changelog"`
}

// PebbleApplicationRelease describes the `release` tag of a pebble JSON
type PebbleApplicationRelease struct {
	Id        string   `json:"id"`
	PbwUrl    string   `json:"pbw_file"`
	Published JSONTime `json:"published_date"`
	Version   string   `json:"version"`
}

// PebbleVersion describes a version change
type PebbleVersion struct {
	Version   string   `json:"version"`
	Published JSONTime `json:"published_date"`
	Notes     string   `json:"release_notes"`
}

// PebbleCompatibility describes the `compatibility` tag of a pebble JSON
type PebbleCompatibility struct {
	Ios     PebbleCompatibilityBool `json:"ios"`
	Android PebbleCompatibilityBool `json:"android"`
	Aplite  PebbleCompatibilityBool `json:"aplite"`
	Basalt  PebbleCompatibilityBool `json:"basalt"`
	Chalk   PebbleCompatibilityBool `json:"chalk"`
	Diorite PebbleCompatibilityBool `json:"diorite"`
}

// PebbleCompatibilityBool describes the contents of a `compatibility` tag of a pebble JSON
type PebbleCompatibilityBool struct {
	Supported bool `json:"supported"`
}

// PebbleHeaderImages is a generic type to allow mixed contents (empty string or array of header images)
type PebbleHeaderImages []PebbleHeaderImage

// PebbleScreenshotImages is a generic type to allow mixed contents (empty string or array of screenshots)
type PebbleScreenshotImages []PebbleScreenshotImage

// PebbleHeaderImage is used by PebbleHeaderImages to allow mixed contents
type PebbleHeaderImage struct {
	Orig string `json:"orig"`
}

// PebbleScreenshotImage is used by PebbleHeaderImages to allow mixed contents
type PebbleScreenshotImage map[string]string

// UnmarshalJSON for PebbleHeaderImages allows for mixed content
func (phi *PebbleHeaderImages) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || b[0] == '"' {
		*phi = make([]PebbleHeaderImage, 0)
		return nil
	}

	return json.Unmarshal(b, (*([]PebbleHeaderImage))(phi))
}

// UnmarshalJSON for PebbleScreenshotImages allows for mixed content
func (psi *PebbleScreenshotImages) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || b[0] == '"' {
		*psi = make([]PebbleScreenshotImage, 0)
		return nil
	}

	return json.Unmarshal(b, (*([]PebbleScreenshotImage))(psi))
}

func parseApp(path string, authors *map[string]int, lastAuthorId *int, collectionNames, collectionColors *map[string]string) (*RebbleApplication, *[]RebbleVersion) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var data = PebbleAppList{}

	err = json.Unmarshal(f, &data)
	if err != nil {
		log.Fatal(err)
	}
	if len(data.Apps) != 1 {
		//log.Println(data)
		//log.Println(data.Data)
		panic("Data is not the size of 1")
	}

	// Create author if it doesn't exist
	if _, ok := (*authors)[data.Apps[0].Author]; !ok {
		(*authors)[data.Apps[0].Author] = *lastAuthorId + 1
		*lastAuthorId = *lastAuthorId + 1
	}

	// Create collection if it doesn't exist
	if _, ok := (*collectionNames)[data.Apps[0].CategoryId]; !ok {
		(*collectionNames)[data.Apps[0].CategoryId] = data.Apps[0].CategoryName
		(*collectionColors)[data.Apps[0].CategoryId] = data.Apps[0].CategoryColor
	}

	app := RebbleApplication{}
	app.AppInfo.Tags = make([]RebbleCollection, 1)
	screenshots := make(([]RebbleScreenshotsPlatform), 0)
	app.Assets.Screenshots = &screenshots

	supportedPlatforms := make([]string, 0)
	if data.Apps[0].Compatibility.Ios.Supported {
		supportedPlatforms = append(supportedPlatforms, "ios")
	}
	if data.Apps[0].Compatibility.Android.Supported {
		supportedPlatforms = append(supportedPlatforms, "android")
	}
	if data.Apps[0].Compatibility.Aplite.Supported {
		supportedPlatforms = append(supportedPlatforms, "aplite")
	}
	if data.Apps[0].Compatibility.Basalt.Supported {
		supportedPlatforms = append(supportedPlatforms, "basalt")
	}
	if data.Apps[0].Compatibility.Chalk.Supported {
		supportedPlatforms = append(supportedPlatforms, "chalk")
	}
	if data.Apps[0].Compatibility.Diorite.Supported {
		supportedPlatforms = append(supportedPlatforms, "diorite")
	}

	app.Id = data.Apps[0].Id
	app.Name = data.Apps[0].Name
	app.AppInfo.Tags[0].Id = data.Apps[0].CategoryId
	app.AppInfo.Tags[0].Name = data.Apps[0].CategoryName
	app.AppInfo.Tags[0].Color = data.Apps[0].CategoryColor
	app.Published = data.Apps[0].Published
	app.Description = data.Apps[0].Description
	app.ThumbsUp = data.Apps[0].Hearts
	app.Type = data.Apps[0].Type
	app.SupportedPlatforms = supportedPlatforms
	app.Author = RebbleAuthor{(*authors)[data.Apps[0].Author], data.Apps[0].Author}
	app.AppInfo.PbwUrl = data.Apps[0].Release.PbwUrl
	app.AppInfo.RebbleReady = false
	app.AppInfo.Updated = data.Apps[0].Release.Published
	app.AppInfo.Version = data.Apps[0].Release.Version
	app.AppInfo.SupportUrl = ""
	app.AppInfo.AuthorUrl = data.Apps[0].Website
	app.AppInfo.SourceUrl = data.Apps[0].Source
	if len(data.Apps[0].HeaderImages) > 0 {
		app.Assets.Banner = data.Apps[0].HeaderImages[0].Orig
	} else {
		app.Assets.Banner = ""
	}
	app.Assets.Icon = ""
	screenshots = append(*app.Assets.Screenshots, RebbleScreenshotsPlatform{data.Apps[0].ScreenshotHardware, make([]string, 0)})
	app.Assets.Screenshots = &screenshots
	for _, screenshot := range data.Apps[0].Screenshots {
		for _, s := range screenshot {
			(*app.Assets.Screenshots)[0].Screenshots = append((*app.Assets.Screenshots)[0].Screenshots, s)
		}
	}
	app.DoomsdayBackup = false

	versions := make([]RebbleVersion, len(data.Apps[0].Changelog))
	for i, pv := range data.Apps[0].Changelog {
		versions[i].Number = pv.Version
		versions[i].Description = pv.Notes
		versions[i].ReleaseDate = pv.Published
	}

	return &app, &versions
}

// HomeHandler is the index page.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile("static/home.html")
	if err != nil {
		log.Fatal("Could not read static/home.html")
	}

	fmt.Fprintf(w, string(data))
}

func RecurseFolder(w http.ResponseWriter, path string, f os.FileInfo, lvl int) {
	for i := 0; i < lvl; i++ {
		w.Write([]byte("="))
	}
	fmt.Fprintf(w, "> %s<br />", f.Name())
	if f.IsDir() {
		fpath := fmt.Sprintf("%s/%s", path, f.Name())
		folder, err := ioutil.ReadDir(fpath)
		if err != nil {
			log.Println(err)
			return
		}
		for _, f1 := range folder {
			RecurseFolder(w, fpath, f1, lvl+1)
		}
	}
}

//var db *sql.DB

// AppsHandler lists all of the available applications from the backend DB.
func AppsHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	db, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to DB"))
		log.Println(err)
		return
	}
	defer db.Close()
	rows, err := db.Query(`
			SELECT apps.name, authors.name
			FROM apps
			JOIN authors ON apps.author_id = authors.id
			ORDER BY published_date ASC LIMIT 20
	`)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to DB"))
		log.Println(err)
		return
	}
	for rows.Next() {
		item := RebbleApplication{}
		err = rows.Scan(&item.Name, &item.Author.Name)
		fmt.Fprintf(w, "Item: %s\n Author: %s\n\n", item.Name, item.Author.Name)
	}
}

// AppHandler returns a particular application from the backend DB as JSON
func AppHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	db, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to db"))
		log.Println(err)
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT apps.id, apps.name, apps.author_id, authors.name, apps.tag_ids, apps.description, apps.thumbs_up, apps.type, apps.supported_platforms, apps.published_date, apps.pbw_url, apps.rebble_ready, apps.updated, apps.version, apps.support_url, apps.author_url, apps.source_url, apps.screenshot_urls, apps.banner_url, apps.icon_url, apps.doomsday_backup FROM apps JOIN authors ON apps.author_id = authors.id WHERE apps.id=?", mux.Vars(r)["id"])
	if err != nil {
		log.Fatal(err)
	}
	exists := rows.Next()
	if !exists {
		w.WriteHeader(404)
		w.Write([]byte("No application with this ID"))
		return
	}

	app := RebbleApplication{}
	var supportedPlatforms_b []byte
	var t_published, t_updated int64
	var tagIds_b []byte
	var tagIds []string
	var screenshots_b []byte
	var screenshots *([]RebbleScreenshotsPlatform)
	err = rows.Scan(&app.Id, &app.Name, &app.Author.Id, &app.Author.Name, &tagIds_b, &app.Description, &app.ThumbsUp, &app.Type, &supportedPlatforms_b, &t_published, &app.AppInfo.PbwUrl, &app.AppInfo.RebbleReady, &t_updated, &app.AppInfo.Version, &app.AppInfo.SupportUrl, &app.AppInfo.AuthorUrl, &app.AppInfo.SourceUrl, &screenshots_b, &app.Assets.Banner, &app.Assets.Icon, &app.DoomsdayBackup)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(supportedPlatforms_b, &app.SupportedPlatforms)
	app.Published.Time = time.Unix(0, t_published)
	app.AppInfo.Updated.Time = time.Unix(0, t_updated)
	json.Unmarshal(tagIds_b, &tagIds)
	app.AppInfo.Tags = make([]RebbleCollection, len(tagIds))
	json.Unmarshal(screenshots_b, &screenshots)
	app.Assets.Screenshots = screenshots

	for i, tagId := range tagIds {
		rows, err := db.Query("SELECT id, name, color FROM collections WHERE id=?", tagId)
		if err != nil {
			log.Fatal(err)
		}

		rows.Next()
		err = rows.Scan(&app.AppInfo.Tags[i].Id, &app.AppInfo.Tags[i].Name, &app.AppInfo.Tags[i].Color)
		if err != nil {
			log.Fatal(err)
		}
	}

	data, err := json.MarshalIndent(app, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

// TagsHandler returns the list of tags of a particular appliction as JSON
func TagsHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	db, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to db"))
		log.Println(err)
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT apps.tag_ids FROM apps")
	if err != nil {
		log.Fatal(err)
	}
	exists := rows.Next()
	if !exists {
		w.WriteHeader(404)
		w.Write([]byte("No application with this ID"))
		return
	}

	var tagIds_b []byte
	var tagIds []string
	err = rows.Scan(&tagIds_b)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(tagIds_b, &tagIds)
	tagList := RebbleTagList{}
	tagList.Tags = make([]RebbleCollection, len(tagIds))

	for i, tagId := range tagIds {
		rows, err := db.Query("SELECT id, name, color FROM collections WHERE id=?", tagId)
		if err != nil {
			log.Fatal(err)
		}

		rows.Next()
		err = rows.Scan(&tagList.Tags[i].Id, &tagList.Tags[i].Name, &tagList.Tags[i].Color)
		if err != nil {
			log.Fatal(err)
		}
	}

	data, err := json.MarshalIndent(tagList, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func VersionsHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	db, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to db"))
		log.Println(err)
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT apps.versions FROM apps")
	if err != nil {
		log.Fatal(err)
	}
	exists := rows.Next()
	if !exists {
		w.WriteHeader(404)
		w.Write([]byte("No application with this ID"))
		return
	}

	var versions_b []byte
	var versions []RebbleVersion
	err = rows.Scan(&versions_b)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(versions_b, &versions)
	changelog := RebbleChangelog{}
	changelog.Versions = versions

	data, err := json.MarshalIndent(changelog, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}

func WriteCommonHeaders(w http.ResponseWriter) {
	// http://stackoverflow.com/a/24818638
	w.Header().Add("Access-Control-Allow-Origin", "http://docs.rebble.io")
	w.Header().Add("Access-Control-Allow-Methods", "GET,POST")
}
