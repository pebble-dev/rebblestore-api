package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"pebble-dev/rebblestore-api/db"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
)

type RebbleCollection struct {
	Id    string          `json:"id"`
	Name  string          `json:"name"`
	Pages int             `json:"pages"`
	Cards []db.RebbleCard `json:"cards"`
}

func insert(apps *([]db.RebbleApplication), location int, app db.RebbleApplication) *([]db.RebbleApplication) {
	beggining := (*apps)[:location]
	end := make([]db.RebbleApplication, len(*apps)-len(beggining))
	copy(end, (*apps)[location:])
	beggining = append(beggining, app)
	beggining = append(beggining, end...)

	return &beggining
}

func remove(apps *([]db.RebbleApplication), location int) *([]db.RebbleApplication) {
	new := make([]db.RebbleApplication, location)
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

func nCompatibleApps(apps *([]db.RebbleApplication), platform string) int {
	var n int
	for _, app := range *apps {
		if platform == "all" || in_array(platform, app.SupportedPlatforms) {
			n = n + 1
		}
	}

	return n
}

type Comparator func(a *db.RebbleApplication, b *db.RebbleApplication) bool

type sorter struct {
	data       *[]db.RebbleApplication
	comparator Comparator
}

func (s sorter) Len() int {
	return len(*s.data)
}

func (s sorter) Swap(i, j int) {
	(*s.data)[i], (*s.data)[j] = (*s.data)[j], (*s.data)[i]
}

func (s sorter) Less(i, j int) bool {
	return s.comparator(&(*s.data)[i], &(*s.data)[j])
}

// PopularFirst is a comparator that sorts applications by ThumbsUp ascending
func PopularFirst(a *db.RebbleApplication, b *db.RebbleApplication) bool {
	return a.ThumbsUp > b.ThumbsUp
}

// PopularLast is a comparator that sorts applications by ThumbsUp descending
func PopularLast(a *db.RebbleApplication, b *db.RebbleApplication) bool {
	return !PopularFirst(a, b)
}

// NewelyPublishedFirst is a comparator that sorts applications by Published date ascending
func NewelyPublishedFirst(a *db.RebbleApplication, b *db.RebbleApplication) bool {
	return a.Published.UnixNano() > b.Published.UnixNano()
}

// OldestPublishedFirst is a comparator that sorts applications by Published date descending
func OldestPublishedFirst(a *db.RebbleApplication, b *db.RebbleApplication) bool {
	return !NewelyPublishedFirst(a, b)
}

func bestApps(apps *([]db.RebbleApplication), sortBy Comparator, nApps int, platform string) *([]db.RebbleApplication) {
	sortedApps := sortApps(apps, sortBy)

	if len(*sortedApps) <= nApps {
		return sortedApps
	}

	nSortedApps := make([]db.RebbleApplication, nApps)
	copy(nSortedApps, *sortedApps)
	return &nSortedApps
}

func sortApps(apps *[]db.RebbleApplication, comparator Comparator) *([]db.RebbleApplication) {
	sortedApps := make([]db.RebbleApplication, len(*apps))
	copy(sortedApps, *apps)
	sort.Sort(sorter{&sortedApps, comparator})
	return &sortedApps
}

// CollectionHandler serves a list of cards from a collection
func CollectionHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	urlquery := r.URL.Query()

	if _, ok := mux.Vars(r)["id"]; !ok {
		return http.StatusBadRequest, errors.New("Missing 'id' parameter")
	}

	sortBy := NewelyPublishedFirst
	if o, ok := urlquery["order"]; ok {
		if len(o) > 1 {
			return http.StatusBadRequest, errors.New("Multiple 'order' parameters are not allowed")
		} else if o[0] == "popular" {
			sortBy = PopularFirst
		} else if o[0] == "new" {
			sortBy = NewelyPublishedFirst
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

	apps, err := ctx.Database.GetAppsForCollection(mux.Vars(r)["id"])
	if err != nil {
		return http.StatusInternalServerError, err
	}
	nCompatibleApps := nCompatibleApps(&apps, platform)
	apps = *(bestApps(&apps, sortBy, page*12, platform))

	collectionName, err := ctx.Database.GetCollectionName(mux.Vars(r)["id"])
	if err != nil {
		return http.StatusInternalServerError, err
	}

	pages := nCompatibleApps / 12
	if nCompatibleApps%12 > 0 {
		pages = pages + 1
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

	if page != 1 && page != pages {
		apps = apps[(page-1)*12 : page*12]
	} else if page == pages {
		apps = apps[(page-1)*12:]
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
