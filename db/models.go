package db

// RebbleCard contains succint information about an app, to display on a result page for example
type RebbleCard struct {
	Id       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	ImageUrl string `json:"image_url"`
	ThumbsUp int    `json:"thumbs_up"`
}

// RebbleCards is a collection of RebbleCard
type RebbleCards struct {
	Cards []RebbleCard `json:"cards"`
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
	Id   string `json:"id"`
	Name string `json:"name"`
}

// RebbleCollection describes the collection (category) of a Rebble application
type RebbleCollection struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
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

// RebbleVersion contains information about a specific version of an app
type RebbleVersion struct {
	Number      string   `json:"number"`
	ReleaseDate JSONTime `json:"release_date"`
	Description string   `json:"description"`
}
