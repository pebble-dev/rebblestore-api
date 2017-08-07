package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// walkFiles is intended to quickly crawl the pebble application folder
// in-order to re-build the application database.
func walkFiles(root string) (<-chan string, <-chan error) {
	// Create a couple of channels to communicate with the main process.
	// (multi-threading FTW!)
	paths := make(chan string)
	errf := make(chan error, 1)

	// Crawl the directory in the background.
	go func() {
		defer close(paths)
		errf <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(info.Name(), ".json") {
				paths <- path
			}
			return nil
		})
	}()

	// Return the channels so that our goroutine can communicate with the main
	// thread.
	return paths, errf
}

// JSONTime is a dummy time object that is meant to allow Go's JSON module to
// properly de-serialize the JSON time format.
type JSONTime struct {
	time.Time
}

// UnmarshalJSON allows for the custom time format within the application JSON
// to be decoded into Go's native time format.
func (self *JSONTime) UnmarshalJSON(b []byte) (err error) {
	s := string(b)

	// Return an empty time.Time object if it didn't exist in the first place.
	if s == "null" {
		self.Time = time.Time{}
		return
	}

	t, err := time.Parse("\"2006-01-02T15:04:05.999Z\"", s)
	if err != nil {
		t = time.Time{}
	}
	self.Time = t
	return
}

// AdminRebuildDBHandler allows an administrator to rebuild the database from
// the application directory after hitting a single API end point.
func AdminRebuildDBHandler(ctx *handlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	//w.WriteHeader(418)
	//fmt.Fprintf(w, "I'm a teapot!")
	/*
		w.Header().Add("content-type", "text/html")
		path, errc := walkFiles("PebbleAppStore/apps")
		for item := range path {
			fmt.Fprintf(w, "File: %s<br />", item)
		}
		if err := <-errc; err != nil {
			log.Fatal(err)
		}
		/**/

	//return /*
	//db.Close()
	os.Remove("./RebbleAppStore.db")
	db, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		log.Fatal(err)
	}

	sqlStmt := `
			create table apps (
				id text not null primary key,
				name text,
				author_id integer,
				category text,
				description text,
				published_date integer,
				pbw_url text,
				rebble_ready integer,
				updated integer,
				version text
			);
			delete from apps;
		`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal("%q: %s\n", err, sqlStmt)
	}

	// Placeholder until we implement an actual author/developer system.
	sqlStmt = `
			create table authors (
				id text not null primary key,
				name text
			);
			delete from authors;
		`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		return 500, fmt.Errorf("%q: %s", err, sqlStmt)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("INSERT INTO apps(id, name, author_id, category, description, published_date, pbw_url, rebble_ready, updated, version) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	authors := make(map[string]int)
	lastId := 0
	path, errc := walkFiles("PebbleAppStore/apps")
	for item := range path {
		//fmt.Fprintf(w, "File: %s<br />", item)
		app := parseApp(item, &authors, &lastId)
		_, err = stmt.Exec(app.Id, app.Name, app.Author.Id, app.Category, app.Description, app.Published.UnixNano(), app.AppInfo.PbwUrl, app.AppInfo.RebbleReady, app.AppInfo.Updated.UnixNano(), app.AppInfo.Version)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := <-errc; err != nil {
		log.Fatal(err)
	}

	for author, id := range authors {
		_, err = tx.Exec("INSERT INTO authors(id, name) VALUES(?, ?)", id, author)
		if err != nil {
			log.Fatal(err)
		}
	}

	tx.Commit()

	log.Print("AppStore Database rebuilt successfully.")
	return 200, nil

}

// AdminVersionHandler returns the latest build information from the host
// in-which it was built on, such as: The current application version, the host
// that built the binary, the date in-which the binary was built, and the
// current git commit hash. Build information is populated during builds
// triggered via the "make build" or "sup production deploy" commands.
func AdminVersionHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	fmt.Fprintf(w, "Version %s\nBuild Host: %s\nBuild Date: %s\nBuild Hash: %s\n", Buildversionstring, Buildhost, Buildstamp, Buildgithash)
}
