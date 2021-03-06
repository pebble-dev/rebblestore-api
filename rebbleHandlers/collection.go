package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"pebble-dev/rebblestore-api/db"
	"strconv"

	"github.com/gorilla/mux"
)

type RebbleCollection struct {
	Id    string          `json:"id"`
	Name  string          `json:"name"`
	Pages int             `json:"pages"`
	Cards []db.RebbleCard `json:"cards"`
}

func in_array(s string, array []string) bool {
	for _, item := range array {
		if item == s {
			return true
		}
	}

	return false
}

func nCompatibleApps(apps *([]db.RebbleApplication), platform string) int {
	var n int
	for _, app := range *apps {
		if platform == "all" || in_array(platform, app.SupportedPlatforms) {
			n = n + 1
		}
	}

	return n
}

// CollectionHandler serves a list of cards from a collection
func CollectionHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	urlquery := r.URL.Query()

	if _, ok := mux.Vars(r)["id"]; !ok {
		return http.StatusBadRequest, errors.New("Missing 'id' parameter")
	}

	var sortByPopular bool
	if o, ok := urlquery["order"]; ok {
		if len(o) > 1 {
			return http.StatusBadRequest, errors.New("Multiple 'order' parameters are not allowed")
		} else if o[0] == "popular" {
			sortByPopular = true
		} else if o[0] == "new" {
			sortByPopular = false
		} else {
			return http.StatusBadRequest, errors.New("Invalid 'order' parameter")
		}
	}
	platform := "all"
	if o, ok := urlquery["platform"]; ok {
		if len(o) > 1 {
			return http.StatusBadRequest, errors.New("Multiple 'platform' parameters are not allowed")
		} else if o[0] == "aplite" || o[0] == "basalt" || o[0] == "chalk" || o[0] == "diorite" {
			platform = o[0]
		} else {
			return http.StatusBadRequest, errors.New("Invalid 'platform' parameter")
		}
	}
	page := 1
	if o, ok := urlquery["page"]; ok {
		if len(o) > 1 {
			return http.StatusBadRequest, errors.New("Multiple pages not allowed")
		} else {
			var err error
			page, err = strconv.Atoi(o[0])
			if err != nil || page < 1 {
				return http.StatusBadRequest, errors.New("Parameter 'page' should be a positive, non-nul integer")
			}
		}
	}

	apps, err := ctx.Database.GetAppsForCollection(mux.Vars(r)["id"], sortByPopular)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	collectionName, err := ctx.Database.GetCollectionName(mux.Vars(r)["id"])
	if err != nil {
		return http.StatusInternalServerError, err
	}

	nCompatibleApps := nCompatibleApps(&apps, platform)
	pages := nCompatibleApps / 12
	if nCompatibleApps%12 > 0 {
		pages = pages + 1
	}

	if page != pages {
		apps = apps[(page-1)*12 : page*12]
	} else if page == pages {
		apps = apps[(page-1)*12:]
	}

	// Only allow to view up to 20 pages - More pages = more computation time
	if pages > 20 {
		pages = 20
	}

	collection := RebbleCollection{
		Id:    mux.Vars(r)["id"],
		Name:  collectionName,
		Pages: pages,
	}

	if page > pages {
		return http.StatusBadRequest, errors.New("Requested inexistant page number")
	}

	for _, app := range apps {
		image := ""
		if len(*app.Assets.Screenshots) != 0 && len((*app.Assets.Screenshots)[0].Screenshots) != 0 {
			image = (*app.Assets.Screenshots)[0].Screenshots[0]
		}
		collection.Cards = append(collection.Cards, db.RebbleCard{
			Id:       app.Id,
			Title:    app.Name,
			Type:     app.Type,
			ImageUrl: image,
			ThumbsUp: app.ThumbsUp,
		})
	}

	data, err := json.MarshalIndent(collection, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)

	return http.StatusOK, nil
}
