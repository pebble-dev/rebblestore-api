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

// A list of RebbleApplications
type PebbleAppList struct {
	Apps []*PebbleApplication `json:"data"`
}

// RebbleApplication contains Pebble App information from the DB
type RebbleApplication struct {
	Id          string        `json:"id"`
	Name        string        `json:"title"`
	Author      RebbleAuthor  `json:"author"`
	Description string        `json:"description"`
	Published   JSONTime      `json:"published_date"`
	AppInfo     RebbleAppInfo `json:"appInfo"`
	Assets      RebbleAssets  `json:"assets"`
}

type RebbleAppInfo struct {
	PbwUrl      string           `json:"pbwUrl"`
	RebbleReady bool             `json:"rebbleReady"`
	Tags        []RebbleCategory `json:"tags"`
	Updated     JSONTime         `json:"updated"`
	Version     string           `json:"version"`
	SupportUrl  string           `json:"supportUrl"`
	AuthorUrl   string           `json:"authorUrl"`
	SourceUrl   string           `json:"sourceUrl"`
}

type RebbleAuthor struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type RebbleAssets struct {
	Banner      string   `json:"appBanner"`
	Icon        string   `json:"appIcon"`
	Screenshots []string `json:"screenshots"`
}

type RebbleCategory struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// PebbleApplication is used by the parseApp() function. It matches directly the `{id}.json` format.
type PebbleApplication struct {
	Id            string                   `json:"id"`
	Name          string                   `json:"title"`
	Author        string                   `json:"author"`
	CategoryId    string                   `json:"category_id"`
	CategoryName  string                   `json:"category_name"`
	CategoryColor string                   `json:"category_color"`
	Description   string                   `json:"description"`
	Published     JSONTime                 `json:"published_date"`
	Release       PebbleApplicationRelease `json:"latest_release"`
	Website       string                   `json:"website"`
	Source        string                   `json:"source"`
	Screenshots   PebbleScreenshotImages   `json:"screenshot_images"`
	HeaderImages  PebbleHeaderImages       `json:"header_images"`
}

type PebbleApplicationRelease struct {
	Id        string   `json:"id"`
	PbwUrl    string   `json:"pbw_file"`
	Published JSONTime `json:"published_date"`
	Version   string   `json:"version"`
}

/*
 * Screenshots and Header images are stored as arrays in the json files. But when there is no screenshot or header, the array is just an empty string, which is fine with dynamically typed languages. However, with Go, we have to redefine UnmarshalJSON for those types.
 */
type PebbleHeaderImages []PebbleHeaderImage
type PebbleScreenshotImages []PebbleScreenshotImage

type PebbleHeaderImage struct {
	Orig string `json:"orig"`
}

type PebbleScreenshotImage struct {
	Screenshot string `json:"144x168"`
}

func (phi *PebbleHeaderImages) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || b[0] == '"' {
		*phi = make([]PebbleHeaderImage, 0)
		return nil
	}

	return json.Unmarshal(b, (*([]PebbleHeaderImage))(phi))
}

func (psi *PebbleScreenshotImages) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || b[0] == '"' {
		*psi = make([]PebbleScreenshotImage, 0)
		return nil
	}

	return json.Unmarshal(b, (*([]PebbleScreenshotImage))(psi))
}

func parseApp(path string, authors *map[string]int, lastAuthorId *int, categoriesNames, categoriesColors *map[string]string) *RebbleApplication {
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

	// Create category if it doesn't exist
	if _, ok := (*categoriesNames)[data.Apps[0].CategoryId]; !ok {
		(*categoriesNames)[data.Apps[0].CategoryId] = data.Apps[0].CategoryName
		(*categoriesColors)[data.Apps[0].CategoryId] = data.Apps[0].CategoryColor
	}

	app := RebbleApplication{}
	app.AppInfo.Tags = make([]RebbleCategory, 1)
	app.Assets.Screenshots = make([]string, 0)
	app.Id = data.Apps[0].Id
	app.Name = data.Apps[0].Name
	app.AppInfo.Tags[0].Id = data.Apps[0].CategoryId
	app.AppInfo.Tags[0].Name = data.Apps[0].CategoryName
	app.AppInfo.Tags[0].Color = data.Apps[0].CategoryColor
	app.Published = data.Apps[0].Published
	app.Description = data.Apps[0].Description
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
	for _, screenshot := range data.Apps[0].Screenshots {
		app.Assets.Screenshots = append(app.Assets.Screenshots, screenshot.Screenshot)
	}

	return &app
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
	rows, err := db.Query("SELECT apps.id, apps.name, apps.author_id, authors.name, apps.tag_ids, apps.description, apps.published_date, apps.pbw_url, apps.rebble_ready, apps.updated, apps.version, apps.support_url, apps.author_url, apps.source_url, apps.screenshot_urls, apps.banner_url, apps.icon_url FROM apps JOIN authors ON apps.author_id = authors.id WHERE apps.id=?", mux.Vars(r)["id"])
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
	var t_published, t_updated int64
	var tagIds_b []byte
	var tagIds []string
	var screenshots_b []byte
	var screenshots []string
	err = rows.Scan(&app.Id, &app.Name, &app.Author.Id, &app.Author.Name, &tagIds_b, &app.Description, &t_published, &app.AppInfo.PbwUrl, &app.AppInfo.RebbleReady, &t_updated, &app.AppInfo.Version, &app.AppInfo.SupportUrl, &app.AppInfo.AuthorUrl, &app.AppInfo.SourceUrl, &screenshots_b, &app.Assets.Banner, &app.Assets.Icon)
	if err != nil {
		log.Fatal(err)
	}
	app.Published.Time = time.Unix(0, t_published)
	app.AppInfo.Updated.Time = time.Unix(0, t_updated)
	json.Unmarshal(tagIds_b, &tagIds)
	app.AppInfo.Tags = make([]RebbleCategory, len(tagIds))
	json.Unmarshal(screenshots_b, &screenshots)

	for i, tagId := range tagIds {
		rows, err := db.Query("SELECT id, name, color FROM categories WHERE id=?", tagId)
		if err != nil {
			log.Fatal(err)
		}

		rows.Next()
		err = rows.Scan(&app.AppInfo.Tags[i].Id, &app.AppInfo.Tags[i].Name, &app.AppInfo.Tags[i].Color)
		if err != nil {
			log.Fatal(err)
		}
	}

	app.Assets.Screenshots = make([]string, len(screenshots))
	for i, screenshot := range screenshots {
		app.Assets.Screenshots[i] = screenshot
	}

	data, err := json.MarshalIndent(app, "", "\t")
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
