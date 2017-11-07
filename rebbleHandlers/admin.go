package rebbleHandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pebble-dev/rebblestore-api/db"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nu7hatch/gouuid"
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

// AdminRebuildDBHandler allows an administrator to rebuild the database from
// the application directory after hitting a single API end point.
func AdminRebuildDBHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
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

	dbHandler := ctx.Database

	// tag_ids and screenshot_urls are Marshaled arrays, hence the BLOB type.
	sqlStmt := `
			drop table if exists apps;
			create table apps (
				id text not null primary key,
				name text,
				author_id integer,
				tag_ids blob,
				description text,
				thumbs_up integer,
				type text,
				supported_platforms blob,
				published_date integer,
				pbw_url text,
				rebble_ready integer,
				updated integer,
				version text,
				support_url text,
				author_url text,
				source_url text,
				screenshots blob,
				banner_url text,
				icon_url text,
				doomsday_backup integer,
				versions blob
			);
			delete from apps;
		`
	_, err := dbHandler.Exec(sqlStmt)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Placeholder until we implement an actual author/developer system.
	sqlStmt = `
			drop table if exists authors;
			create table authors (
				id text not null primary key,
				name text
			);
			delete from authors;
		`
	_, err = dbHandler.Exec(sqlStmt)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("%q: %s", err, sqlStmt)
	}

	// Placeholder until we implement an actual collections system.
	sqlStmt = `
			drop table if exists collections;
			create table collections (
				id text not null primary key,
				name text,
				color text,
				apps blob,
				cache_apps_most_popular blob,
				cache_time integer
			);
			delete from collections;
		`
	_, err = dbHandler.Exec(sqlStmt)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("%q: %s", err, sqlStmt)
	}

	tx, err := dbHandler.Begin()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO apps(id, name, author_id, tag_ids, description, thumbs_up, type, supported_platforms, published_date, pbw_url, rebble_ready, updated, version, support_url, author_url, source_url, screenshots, banner_url, icon_url, doomsday_backup, versions) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer stmt.Close()

	authors := make(map[string]int)
	collections := make(map[string]db.RebbleCollection)
	lastAuthorId := 0
	path, errc := walkFiles("PebbleAppStore/")
	apps := make(map[string]db.RebbleApplication)
	versions := make(map[string]([]db.RebbleVersion))
	for item := range path {
		app, v, err := parseApp(item, &authors, &lastAuthorId, &collections)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		if _, ok := apps[app.Id]; ok {
			platformExists := false
			for _, platform := range *apps[app.Id].Assets.Screenshots {
				if platform.Platform == (*app.Assets.Screenshots)[0].Platform {
					platformExists = true
				}
			}

			if !platformExists {
				(*apps[app.Id].Assets.Screenshots) = append((*apps[app.Id].Assets.Screenshots), (*app.Assets.Screenshots)[0])
			}
		} else {
			apps[app.Id] = *app
			versions[app.Id] = *v
		}
	}

	for _, app := range apps {
		tag_ids_s := make([]string, 1)
		tag_ids_s[0] = app.AppInfo.Tags[0].Id
		tag_ids, err := json.Marshal(tag_ids_s)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		screenshots, err := json.Marshal(app.Assets.Screenshots)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		supported_platforms, err := json.Marshal(app.SupportedPlatforms)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		versions, err := json.Marshal(versions[app.Id])
		if err != nil {
			return http.StatusInternalServerError, err
		}

		_, err = stmt.Exec(app.Id, app.Name, app.Author.Id, tag_ids, app.Description, app.ThumbsUp, app.Type, supported_platforms, app.Published.UnixNano(), app.AppInfo.PbwUrl, app.AppInfo.RebbleReady, app.AppInfo.Updated.UnixNano(), app.AppInfo.Version, app.AppInfo.SupportUrl, app.AppInfo.AuthorUrl, app.AppInfo.SourceUrl, screenshots, app.Assets.Banner, app.Assets.Icon, app.DoomsdayBackup, versions)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}
	if err := <-errc; err != nil {
		return http.StatusInternalServerError, err
	}

	for author, id := range authors {
		_, err = tx.Exec("INSERT INTO authors(id, name) VALUES(?, ?)", id, author)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	for id, collection := range collections {
		collectionApps := make([]string, 0)
		for _, app := range apps {
			for _, tag := range app.AppInfo.Tags {
				if tag.Id == collection.Id {
					collectionApps = append(collectionApps, app.Id)
				}
			}
		}

		collectionApps_b, err := json.Marshal(collectionApps)

		_, err = tx.Exec("INSERT INTO collections(id, name, color, apps) VALUES(?, ?, ?, ?)", id, collection.Name, collection.Color, collectionApps_b)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	tx.Commit()

	log.Print("AppStore Database rebuilt successfully.")
	return http.StatusOK, nil

}

// AdminRebuildImagesHandler allows an administrator to rebuild the images database from the application directory after hitting a single API end point.
func AdminRebuildImagesHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	dbHandler := ctx.Database

	err := os.RemoveAll("PebbleImages")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = os.Mkdir("PebbleImages", 0755)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	tx, err := dbHandler.Begin()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	rows, err := tx.Query("SELECT id, screenshots FROM apps")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	apps := make(map[string]db.RebbleApplication, 0)
	urls := make([]string, 0)
	for rows.Next() {
		var id string
		var screenshots_b []byte
		var screenshots *([]db.RebbleScreenshotsPlatform)
		err = rows.Scan(&id, &screenshots_b)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		err = json.Unmarshal(screenshots_b, &screenshots)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		platforms := make([]db.RebbleScreenshotsPlatform, 0)
		apps[id] = db.RebbleApplication{
			Id: id,
			Assets: db.RebbleAssets{
				Screenshots: &platforms,
			},
		}

		for _, platform := range *screenshots {
			newPlatform := db.RebbleScreenshotsPlatform{
				Platform: platform.Platform,
			}
			errs := make([](chan error), 0)

			for _, screenshot := range platform.Screenshots {
				if in_array(screenshot, urls) {
					continue
				}

				u, err := uuid.NewV4()
				if err != nil {
					return http.StatusInternalServerError, err
				}

				newPlatform.Screenshots = append(newPlatform.Screenshots, fmt.Sprintf("/images/%v", u.String()))
				urls = append(urls, screenshot)

				errs = append(errs, make(chan error))

				go func(url string, id string, err chan error) {
					log.Println("Downloading", url)
					resp, e := http.Get(url)
					if e != nil {
						err <- e
						return
					}

					out, e := os.Create(fmt.Sprintf("PebbleImages/%v", id))
					if e != nil {
						err <- e
						return
					}

					defer resp.Body.Close()
					_, e = io.Copy(out, resp.Body)

					err <- e
				}(screenshot, u.String(), errs[len(errs)-1])
			}

			for _, err := range errs {
				e := <-err
				if e != nil {
					return http.StatusInternalServerError, e
				}
			}

			if len(newPlatform.Screenshots) > 0 {
				platforms = append(platforms, newPlatform)
			}
		}
	}

	for id, app := range apps {
		screenshots, err := json.Marshal(*app.Assets.Screenshots)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		_, err = tx.Exec("UPDATE apps SET screenshots=? WHERE id=?", screenshots, id)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	tx.Commit()

	log.Print("AppStore Image Database rebuilt successfully.")
	return http.StatusOK, nil

}
