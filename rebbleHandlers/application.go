package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"pebble-dev/rebblestore-api/db"

	"github.com/gorilla/mux"
)

// PebbleAppList contains a list of PebbleApplication. It matches the format of Pebble API answers.
type PebbleAppList struct {
	Apps []*PebbleApplication `json:"data"`
}

// RebbleTagList contains a list of tag. Used by getApi(id)
type RebbleTagList struct {
	Tags []db.RebbleCollection `json:"tags"`
}

// RebbleChangelog contains a list of version changes for an app
type RebbleChangelog struct {
	Versions []db.RebbleVersion `json:"versions"`
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
	Published          db.JSONTime              `json:"published_date"`
	Release            PebbleApplicationRelease `json:"latest_release"`
	Website            string                   `json:"website"`
	Source             string                   `json:"source"`
	Screenshots        PebbleScreenshotImages   `json:"screenshot_images"`
	Icons              PebbleIcons              `json:"icon_image"`
	ScreenshotHardware string                   `json:"screenshot_hardware"`
	HeaderImages       PebbleHeaderImages       `json:"header_images"`
	Hearts             int                      `json:"hearts"`
	Type               string                   `json:"type"`
	Compatibility      PebbleCompatibility      `json:"compatibility"`
	Changelog          []PebbleVersion          `json:"changelog"`
}

// PebbleApplicationRelease describes the `release` tag of a pebble JSON
type PebbleApplicationRelease struct {
	Id        string      `json:"id"`
	PbwUrl    string      `json:"pbw_file"`
	Published db.JSONTime `json:"published_date"`
	Version   string      `json:"version"`
}

// PebbleVersion describes a version change
type PebbleVersion struct {
	Version   string      `json:"version"`
	Published db.JSONTime `json:"published_date"`
	Notes     string      `json:"release_notes"`
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

// PebbleIcons contains the icon at varying resolutions
type PebbleIcons map[string]string

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

func (pi *PebbleIcons) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || b[0] == '"' {
		*pi = make(map[string]string, 0)
		return nil
	}

	return json.Unmarshal(b, (*(map[string]string))(pi))
}

func parseApp(path string, users *map[string]int, lastAuthorId *int, collections *map[string]db.RebbleCollection) (*db.RebbleApplication, *[]db.RebbleVersion, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var data = PebbleAppList{}

	err = json.Unmarshal(f, &data)
	if err != nil {
		log.Print("Error parsing app JSON: " + path)
		return nil, nil, err
	}
	if len(data.Apps) != 1 {
		//log.Println(data)
		//log.Println(data.Data)
		panic("Data is not the size of 1")
	}

	// Create author if it doesn't exist
	if _, ok := (*users)[data.Apps[0].Author]; !ok {
		(*users)[data.Apps[0].Author] = *lastAuthorId + 1
		*lastAuthorId = *lastAuthorId + 1
	}

	// Create collection if it doesn't exist
	if _, ok := (*collections)[data.Apps[0].CategoryId]; !ok {
		(*collections)[data.Apps[0].CategoryId] = db.RebbleCollection{
			Id:    data.Apps[0].CategoryId,
			Name:  data.Apps[0].CategoryName,
			Color: data.Apps[0].CategoryColor,
		}
	}

	app := db.RebbleApplication{}
	app.AppInfo.Tags = make([]db.RebbleCollection, 1)
	screenshots := make(([]db.RebbleScreenshotsPlatform), 0)
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
	app.Author = db.RebbleAuthor{(*users)[data.Apps[0].Author], data.Apps[0].Author}
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
	if icon, ok := data.Apps[0].Icons["48x48"]; ok {
		app.Assets.Icon = icon
	}
	screenshots = append(*app.Assets.Screenshots, db.RebbleScreenshotsPlatform{data.Apps[0].ScreenshotHardware, make([]string, 0)})
	app.Assets.Screenshots = &screenshots
	for _, screenshot := range data.Apps[0].Screenshots {
		for _, s := range screenshot {
			(*app.Assets.Screenshots)[0].Screenshots = append((*app.Assets.Screenshots)[0].Screenshots, s)
		}
	}
	app.DoomsdayBackup = false

	versions := make([]db.RebbleVersion, len(data.Apps[0].Changelog))
	for i, pv := range data.Apps[0].Changelog {
		versions[i].Number = pv.Version
		versions[i].Description = pv.Notes
		versions[i].ReleaseDate = pv.Published
	}

	return &app, &versions, nil
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

// AppsHandler lists all of the available applications from the backend DB.
func AppsHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	page, err := strconv.Atoi(mux.Vars(r)["page"])
	if err != nil {
		return http.StatusBadRequest, err
	}

	urlquery := r.URL.Query()

	var limit int
	var ascending bool
	var sortby string

	if l, ok := urlquery["limit"]; ok {
		if len(l) > 1 {
			return http.StatusBadRequest, errors.New("Multiple 'limit' parameters are not allowed")
		}

		limit, err = strconv.Atoi(l[0])
		if err != nil {
			return http.StatusBadRequest, errors.New("Specified 'limit' parameter is not a parsable integer")
		}

		if limit > 50 {
			return http.StatusBadRequest, errors.New("Specified 'limit' parameter is above the maximum allowed")
		}
	} else {
		limit = 20
	}

	if o, ok := urlquery["order"]; ok {
		if len(o) > 1 {
			return http.StatusBadRequest, errors.New("Multiple 'order' parameters are not allowed")
		} else if o[0] == "asc" {
			ascending = true
		} else if o[0] == "desc" {
			ascending = false
		} else {
			return http.StatusBadRequest, errors.New("Invalid 'order' parameter")
		}
	} else {
		ascending = false
	}

	if sb, ok := urlquery["sortby"]; ok {
		if len(sb) > 1 {
			return http.StatusBadRequest, errors.New("Multiple 'sortby' parameters are not allowed")
		} else if sb[0] == "popular" {
			sortby = sb[0]
		} else if sb[0] == "recent" {
			sortby = sb[0]
		}
	} else {
		sortby = "recent"
	}

	apps, err := ctx.Database.GetAllApps(sortby, ascending, (page-1)*limit, limit)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	data, err := json.Marshal(apps)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Add("content-type", "application/json")
	w.Write(data)
	return http.StatusOK, nil
}

// AppHandler returns a particular application from the backend DB as JSON
func AppHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	app, err := ctx.Database.GetApp(mux.Vars(r)["id"])
	if err != nil {
		return http.StatusInternalServerError, err
	}

	data, err := json.MarshalIndent(app, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
	return http.StatusOK, nil
}

// TagsHandler returns the list of tags of a particular appliction as JSON
func TagsHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	collections, err := ctx.Database.GetAppTags(mux.Vars(r)["id"])
	if err != nil {
		return http.StatusInternalServerError, err
	}

	tagList := RebbleTagList{
		Tags: collections,
	}

	data, err := json.MarshalIndent(tagList, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)

	return http.StatusOK, nil
}

// VersionsHandler returns the server version
func VersionsHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	versions, err := ctx.Database.GetAppVersions(mux.Vars(r)["id"])

	changelog := RebbleChangelog{}
	changelog.Versions = versions

	data, err := json.MarshalIndent(changelog, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)

	return http.StatusOK, nil
}
