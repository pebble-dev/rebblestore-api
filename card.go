package main

// RebbleCard contains succint information about an app, to display on a result page for example
type RebbleCard struct {
	Id        string   `json:"id"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	ImageUrl  string   `json:"image_url"`
	ThumbsUp  int      `json:"thumbs_up"`
	Published JSONTime `json:"published_date"`
}

// RebbleCards is a collection of RebbleCard
type RebbleCards struct {
	Cards []RebbleCard `json:"cards"`
}
