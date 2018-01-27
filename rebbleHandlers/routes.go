package rebbleHandlers

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
func Handlers(context *HandlerContext) *mux.Router {
	r := mux.NewRouter()
	r.Handle("/", routeHandler{context, HomeHandler}).Methods("GET")
	r.Handle("/dev/apps/get_apps/page/{page:[0-9]+}", routeHandler{context, AppsHandler}).Methods("GET")
	r.Handle("/dev/apps/get_app/id/{id}", routeHandler{context, AppHandler}).Methods("GET")
	r.Handle("/dev/apps/get_tags/id/{id}", routeHandler{context, TagsHandler}).Methods("GET")
	r.Handle("/dev/apps/get_versions/id/{id}", routeHandler{context, VersionsHandler}).Methods("GET")
	r.Handle("/dev/apps/get_collection/id/{id}", routeHandler{context, CollectionHandler}).Methods("GET")
	r.Handle("/dev/apps/search/{query}", routeHandler{context, SearchHandler}).Methods("GET")
	r.Handle("/dev/author/id/{id}", routeHandler{context, AuthorHandler}).Methods("GET")
	r.Handle("/user/login", routeHandler{context, AccountLoginHandler}).Methods("POST")
	r.Handle("/user/info", routeHandler{context, AccountInfoHandler}).Methods("POST")
	r.Handle("/user/update/name", routeHandler{context, AccountUpdateNameHandler}).Methods("POST")
	r.Handle("/admin/rebuild/db", routeHandler{context, AdminRebuildDBHandler}).Host("localhost")
	r.Handle("/admin/rebuild/images", routeHandler{context, AdminRebuildImagesHandler}).Host("localhost")
	r.Handle("/admin/version", routeHandler{context, AdminVersionHandler})
	//r.HandleFunc("/boot/{path:.*}", BootHandler).Methods("GET")
	// Added OS parameter
	r.Handle("/boot/{os}/{path:.*}", routeHandler{context, BootHandler}).Methods("GET")
	r.Handle("/images/{image}", routeHandler{context, ImagesHandler}).Methods("GET")

	// The boot parameter wasn't working when set to /boot/ and this was used as
	// an alternative. However, using the {path:.*} matching appears to have
	// solved this issue. For future reference, the below snippet has been left,
	// commented out.
	//r.PathPrefix("/boot/").Methods("GET").HandlerFunc(BootHandler)
	return r
}
