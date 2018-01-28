package rebbleHandlers

import (
	"encoding/json"
	"net/http"
)

type status_ssos struct {
	Ssos []status_sso `json:"ssos"`
}

type status_sso struct {
	Name        string `json:"name"`
	ClientID    string `json:"client_id"`
	DiscoverURI string `json:"discover_uri"`
}

// ClientIdsHandler returns the list of client IDs for the frontend to use
func ClientIdsHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	ssos := status_ssos{
		Ssos: make([]status_sso, len(ctx.SSos)),
	}

	for i, sso := range ctx.SSos {
		ssos.Ssos[i].Name = sso.Name
		ssos.Ssos[i].ClientID = sso.ClientID
		ssos.Ssos[i].DiscoverURI = sso.DiscoverURI
	}

	data, err := json.MarshalIndent(ssos, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
	return http.StatusOK, nil
}
