package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// DummyHandler is useful for adding a route when the handler hasn't been
// completed/fleshed out yet.
func DummyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(501)
	w.Write([]byte("501 Not Implemented\n"))
	fmt.Fprintf(w, "%s?%s\n%#v", r.URL.Path, r.URL.RawQuery, mux.Vars(r))
}

// Handlers returns a mux.Router with all possible routes already setup.
func Handlers(context *handlerContext) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler).Methods("GET")
	r.HandleFunc("/dev/apps", AppsHandler).Methods("GET")
	r.HandleFunc("/dev/apps/get_app/id/{id}", AppHandler).Methods("GET")
	r.HandleFunc("/dev/apps/get_tags/id/{id}", TagsHandler).Methods("GET")
	r.HandleFunc("/dev/apps/get_versions/id/{id}", VersionsHandler).Methods("GET")
	r.HandleFunc("/dev/apps/search/{query}", SearchHandler).Methods("GET")
	r.Handle("/admin/rebuild/db", routeHandler{context, AdminRebuildDBHandler}).Host("localhost")
	r.HandleFunc("/admin/version", AdminVersionHandler)
	//r.HandleFunc("/boot/{path:.*}", BootHandler).Methods("GET")
	// Added OS parameter
	r.HandleFunc("/boot/{os}/{path:.*}", BootHandler).Methods("GET")

	// The boot parameter wasn't working when set to /boot/ and this was used as
	// an alternative. However, using the {path:.*} matching appears to have
	// solved this issue. For future reference, the below snippet has been left,
	// commented out.
	//r.PathPrefix("/boot/").Methods("GET").HandlerFunc(BootHandler)
	return r
}
