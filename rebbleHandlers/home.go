package rebbleHandlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// HomeHandler is the index page.
func HomeHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	data, err := ioutil.ReadFile("static/home.html")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	fmt.Fprintf(w, string(data))

	return http.StatusOK, nil
}
