package rebbleHandlers

import (
	"fmt"
	"net/http"

	"github.com/pebble-dev/rebblestore-api/common"
)

// AdminVersionHandler returns the latest build information from the host
// in-which it was built on, such as: The current application version, the host
// that built the binary, the date in-which the binary was built, and the
// current git commit hash. Build information is populated during builds
// triggered via the "make build" or "sup production deploy" commands.
func AdminVersionHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	fmt.Fprintf(w, "Version %s\nBuild Host: %s\nBuild Date: %s\nBuild Hash: %s\n", common.Buildversionstring, common.Buildhost, common.Buildstamp, common.Buildgithash)

	return http.StatusOK, nil
}
