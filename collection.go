package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func insert(apps *([]RebbleApplication), location int, app RebbleApplication) *([]RebbleApplication) {
	beggining := (*apps)[:location]
	end := make([]RebbleApplication, len(*apps)-len(beggining))
	copy(end, (*apps)[location:])
	beggining = append(beggining, app)
	beggining = append(beggining, end...)

	return &beggining
}

func remove(apps *([]RebbleApplication), location int) *([]RebbleApplication) {
	new := make([]RebbleApplication, location)
	copy(new, (*apps)[:location])
	new = append(new, (*apps)[location+1:]...)

	return &new
}

func in_array(s string, array []string) bool {
	for _, item := range array {
		if item == s {
			return true
		}
	}

	return false
}

func bestApps(apps *([]RebbleApplication), sortByPopular bool, nApps int, platform string) *([]RebbleApplication) {
	newApps := make([]RebbleApplication, 0)

	for _, app := range *apps {
		if platform == "all" || in_array(platform, app.SupportedPlatforms) {
			newApps = append(newApps, app)
		}

		if len(newApps) > nApps {
			if sortByPopular {
				worst := 0
				for i, newApp := range newApps {
					if newApp.ThumbsUp < newApps[worst].ThumbsUp {
						worst = i
					}
				}
				newApps = *(remove(&newApps, worst))
			} else {
				worst := 0
				for i, newApp := range newApps {
					if newApp.Published.UnixNano() < newApps[worst].Published.UnixNano() {
						worst = i
					}
				}
				newApps = *(remove(&newApps, worst))
			}
		}
	}

	return &newApps
}

func sortApps(apps *([]RebbleApplication), sortByPopular bool) *([]RebbleApplication) {
	newApps := make([]RebbleApplication, 0)

	for _, app := range *apps {
		if len(newApps) == 0 {
			newApps = []RebbleApplication{app}

			continue
		} else if len(newApps) == 1 {
			if sortByPopular {
				if newApps[0].ThumbsUp > app.ThumbsUp {
					newApps = []RebbleApplication{newApps[0], app}
				} else {
					newApps = []RebbleApplication{app, newApps[0]}
				}
			} else {
				if newApps[0].Published.UnixNano() > app.Published.UnixNano() {
					newApps = []RebbleApplication{app, newApps[0]}
				} else {
					newApps = []RebbleApplication{newApps[0], app}
				}
			}

			continue
		}

		if sortByPopular {
			added := false
			for i, newApp := range newApps {
				if newApp.ThumbsUp < app.ThumbsUp {
					newApps = *(insert(&newApps, i, app))
					added = true
					break
				}
			}
			if !added {
				newApps = *(insert(&newApps, len(newApps), app))
			}
		} else {
			added := false
			for i, newApp := range newApps {
				if app.Published.UnixNano() > newApp.Published.UnixNano() {
					newApps = *(insert(&newApps, i, app))
					added = true
					break
				}
			}
			if !added {
				newApps = *(insert(&newApps, len(newApps), app))
			}
		}
	}

	return &newApps
}

// CollectionHandler serves a list of cards from a collection
func CollectionHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	db, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to DB"))
		log.Println(err)
		return
	}
	defer db.Close()

	urlquery := r.URL.Query()

	if _, ok := mux.Vars(r)["id"]; !ok {
		w.WriteHeader(400)
		w.Write([]byte("Missing id parameter"))
		return
	}

	var sortByPopular bool
	if o, ok := urlquery["order"]; ok {
		if len(o) > 1 {
			w.WriteHeader(400)
			w.Write([]byte("Multiple order types are not allowed"))
			return
		} else if o[0] == "popular" {
			sortByPopular = true
		} else if o[0] == "new" {
			sortByPopular = false
		} else {
			w.WriteHeader(400)
			w.Write([]byte("Invalid order parameter"))
			return
		}
	}
	platform := "all"
	if o, ok := urlquery["platform"]; ok {
		if len(o) > 1 {
			w.WriteHeader(400)
			w.Write([]byte("Multiple order types are not allowed"))
			return
		} else if o[0] == "aplite" || o[0] == "basalt" || o[0] == "chalk" || o[0] == "diorite" {
			platform = o[0]
		} else {
			w.WriteHeader(400)
			w.Write([]byte("Invalid platform parameter"))
			return
		}
	}
	page := 1
	if o, ok := urlquery["page"]; ok {
		if len(o) > 1 {
			w.WriteHeader(400)
			w.Write([]byte("Multiple pages are not allowed"))
			return
		} else {
			page, err = strconv.Atoi(o[0])
			if err != nil || page < 1 {
				w.WriteHeader(400)
				w.Write([]byte("Page should be a positive, non-nul integer"))
				return
			}
		}
	}

	rows, err := db.Query("SELECT apps FROM collections WHERE id=?", mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Unable to connect to DB"))
		log.Println(err)
		return
	}
	if !rows.Next() {
		w.WriteHeader(500)
		w.Write([]byte("Specified collection does not exist"))
		return
	}
	var appIds_b []byte
	var appIds []string
	err = rows.Scan(&appIds_b)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(appIds_b, &appIds)

	apps := make([]RebbleApplication, 0)
	for _, id := range appIds {
		rows, err = db.Query("SELECT id, name, type, thumbs_up, icon_url, published_date, supported_platforms FROM apps WHERE id=?", id)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Unable to connect to DB"))
			log.Println(err)
			return
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

	apps = *(bestApps(&apps, sortByPopular, page*12, platform))
	apps = *(sortApps(&apps, sortByPopular))
	if page != 1 {
		apps = apps[(page-1)*12 : page*12]
	}

	var cards RebbleCards
	for _, app := range apps {
		cards.Cards = append(cards.Cards, RebbleCard{
			Id:       app.Id,
			Title:    app.Name,
			Type:     app.Type,
			ImageUrl: app.Assets.Icon,
			ThumbsUp: app.ThumbsUp,
		})
	}

	data, err := json.MarshalIndent(cards, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}
