package rebbleHandlers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
)

// ImagesHandler serves a static image from PebbleImages/
func ImagesHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {

	if _, ok := mux.Vars(r)["image"]; !ok {
		return http.StatusForbidden, errors.New("Forbidden")
	}

	// We make sure all characters belong in a UUID to prevent any funky stuff (like trying to access images/../../etc/passwd)
	for _, c := range mux.Vars(r)["image"] {
		if !((c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || (c >= '0' && c <= '9') || c == '-') {
			return http.StatusNotFound, errors.New("File not found")
		}
	}

	data, err := ioutil.ReadFile(fmt.Sprintf("PebbleImages/%v", mux.Vars(r)["image"]))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Write(data)

	return http.StatusOK, nil
}
