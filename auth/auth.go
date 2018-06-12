package auth

import (
	"encoding/json"
	"net/http"
)

type AuthService struct {
	Auth string
}

type name struct {
	Name         string `json:"name"`
	ErrorMessage string `json:"errorMessage"`
}

// GetName gets a user name for a given id
// returns user name, and error message + error if it failed
func (authService *AuthService) GetName(id string) (string, string, error) {
	resp, err := http.Get(authService.Auth + "/user/name/" + id)
	if err != nil {
		return "", "Internal server error: Could not contact authentication server", err
	}

	decoder := json.NewDecoder(resp.Body)
	var name name
	err = decoder.Decode(&name)
	if err != nil {
		return "", "Internal server error: Could not parse authentication server answer", err
	}
	defer resp.Body.Close()

	return name.Name, name.ErrorMessage, nil
}
