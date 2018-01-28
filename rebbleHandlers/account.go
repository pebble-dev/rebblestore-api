package rebbleHandlers

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

// Sso is a JSON object containing information about a specific OpenID SSO provider
type Sso struct {
	Name         string `json:"name"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	DiscoverURI  string `json:"discover_uri"`
	RedirectURI  string `json:"redirect_uri"`

	Discovery Discovery
	certs     certs
}

// Discovery lists all the API endpoints for a given SSO
// https://developers.google.com/identity/protocols/OpenIDConnect#discovery
// Only the relevant fields will be filled
type Discovery struct {
	TokenEndpoint    string `json:"token_endpoint"`
	UserinfoEndpoint string `json:"userinfo_endpoint"`
	JwksURI          string `json:"jwks_uri"`
}

type key struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type certs struct {
	Keys []key `json:"keys"`
}

// This is the response from the exchange of the authorization code for access and ID tokens
type tokensStatus struct {
	AccessToken string `json:"access_token"`
	IdToken     string `json:"id_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`

	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type accountLogin struct {
	Code         string `json:"code"`
	AuthProvider string `json:"authProvider"`
}

type accountLoginStatus struct {
	SessionKey   string `json:"sessionKey"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
}

type updateAccount struct {
	SessionKey string `json:"sessionKey"`
	Name       string `json:"name"`
}

type updateAccountStatus struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
}

type auth struct {
	SessionKey string `json:"sessionKey"`
}

type accountInfo struct {
	LoggedIn     bool   `json:"loggedIn"`
	Name         string `json:"name"`
	ErrorMessage string `json:"errorMessage"`
}

func (key *key) getPublicKey() (rsa.PublicKey, error) {
	// Pad the base64 data if necessary
	if len(key.N)%4 != 0 {
		key.N = key.N + strings.Repeat("=", 4-(len(key.N)%4))
	}
	if len(key.E)%4 != 0 {
		key.E = key.E + strings.Repeat("=", 4-(len(key.E)%4))
	}

	nb := make([]byte, base64.URLEncoding.DecodedLen(len([]byte(key.N))))
	eb := make([]byte, base64.URLEncoding.DecodedLen(len([]byte(key.E))))
	n := big.NewInt(0)
	e := 0
	ln, err := base64.URLEncoding.Decode(nb, []byte(key.N))
	if err != nil {
		return rsa.PublicKey{}, err
	}
	le, err := base64.URLEncoding.Decode(eb, []byte(key.E))
	if err != nil {
		return rsa.PublicKey{}, err
	}

	i := 0
	for i < ln {
		z := big.NewInt(int64(nb[i]))
		z.Lsh(z, 8*uint(ln-i-1))
		n.Add(n, z)
		i++
	}
	i = 0
	for i < le {
		e += int(uint64(eb[i]) << (8 * uint64(le-i-1)))
		i++
	}

	return rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

func accountLoginFail(message string, err error, w *http.ResponseWriter) error {
	status := accountLoginStatus{
		Success:      false,
		ErrorMessage: message,
	}

	data, e := json.MarshalIndent(status, "", "\t")
	if e != nil {
		return e
	}

	// Send the JSON object back to the user
	(*w).Header().Add("content-type", "application/json")
	(*w).Write(data)

	log.Println(err)

	return nil
}

func findKey(sso Sso, kid string) (key, error) {
	foundKey := false
	var key key
	for _, k := range sso.certs.Keys {
		if k.Kid == kid {
			foundKey = true
			key = k
			break
		}
	}

	if foundKey {
		return key, nil
	}

	return key, errors.New("Key not found")
}

func parseJwtToken(sso Sso, encryptedToken string) (jwt.MapClaims, error) {
	resp, err := http.Get(sso.Discovery.JwksURI)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, err
	}

	token, err := jwt.Parse(encryptedToken, func(token *jwt.Token) (interface{}, error) {
		key, err := findKey(sso, token.Header["kid"].(string))

		// We didn't found a suitable decryption key, but it might just be because they have been updated (should happen about once a day)
		if err != nil {
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&sso.certs)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			key, err = findKey(sso, token.Header["kid"].(string))
			if err != nil {
				return nil, errors.New("Could not find suitable decryption key for JWT token")
			}
		}

		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("Expected RSA signing method for JWT token")
		}

		pub, err := key.getPublicKey()
		if err != nil {
			return nil, err
		}

		return &pub, nil
	})

	if err != nil {
		return nil, err
	}

	return token.Claims.(jwt.MapClaims), nil
}

// AccountLoginHandler handles the login of a user
func AccountLoginHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)

	var loginInformation accountLogin
	err := decoder.Decode(&loginInformation)
	if err != nil {
		return http.StatusBadRequest, accountLoginFail("Internal server error: Server could not parse message", err, &w)
	}
	defer r.Body.Close()

	var sso Sso
	foundSso := false
	for _, s := range ctx.SSos {
		if s.Name == loginInformation.AuthProvider {
			sso = s
			foundSso = true
		}
	}

	if !foundSso {
		return http.StatusBadRequest, accountLoginFail("Invalid SSO provider", errors.New("Invalid SSO provider"), &w)
	}

	v := url.Values{}
	v.Add("code", loginInformation.Code)
	v.Add("client_id", sso.ClientID)
	v.Add("client_secret", sso.ClientSecret)
	v.Add("redirect_uri", sso.RedirectURI)
	v.Add("grant_type", "authorization_code")
	resp, err := http.Post(sso.Discovery.TokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(v.Encode()))
	if err != nil {
		return http.StatusInternalServerError, accountLoginFail("Internal server error: Could not exchange tokens", err, &w)
	}

	decoder = json.NewDecoder(resp.Body)
	var tokensStatus tokensStatus
	err = decoder.Decode(&tokensStatus)
	if err != nil {
		return http.StatusInternalServerError, accountLoginFail("Internal server error: Could not decode token information", err, &w)
	}
	defer resp.Body.Close()

	if tokensStatus.Error != "" {
		return http.StatusInternalServerError, accountLoginFail("Internal server error: Could not exchange tokens", errors.New("Could not exchange tokens: "+tokensStatus.Error+" ("+tokensStatus.ErrorDescription+")"), &w)
	}

	claims, err := parseJwtToken(sso, tokensStatus.IdToken)
	if err != nil {
		return http.StatusInternalServerError, accountLoginFail("Internal server error: Could not decode token information", err, &w)
	}

	var name string
	if n, ok := claims["name"]; ok {
		name = n.(string)
	}

	sessionKey, userErr, err := ctx.Database.AccountLoginOrRegister(sso.Name, claims["sub"].(string), name, tokensStatus.AccessToken, int64(claims["exp"].(float64)), r.RemoteAddr)
	if err != nil {
		return http.StatusBadRequest, accountLoginFail(userErr, err, &w)
	}

	status := accountLoginStatus{
		Success:      true,
		ErrorMessage: userErr,
		SessionKey:   sessionKey,
	}
	data, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
	return http.StatusOK, nil
}

// AccountInfoHandler displays the account information for a given Session Key
func AccountInfoHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)

	var auth auth
	err := decoder.Decode(&auth)
	if err != nil {
		return http.StatusBadRequest, err
	}
	defer r.Body.Close()

	loggedIn, errorMessage, err := ctx.Database.SessionInformation(auth.SessionKey)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if !loggedIn {
		info := accountInfo{
			LoggedIn:     false,
			Name:         "",
			ErrorMessage: errorMessage,
		}
		data, err := json.MarshalIndent(info, "", "\t")
		if err != nil {
			return http.StatusInternalServerError, err
		}

		// Send the JSON object back to the user
		w.Header().Add("content-type", "application/json")
		w.Write(data)
		return http.StatusOK, nil
	}

	loggedIn, name, err := ctx.Database.AccountInformation(auth.SessionKey)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	info := accountInfo{
		LoggedIn:     loggedIn,
		Name:         name,
		ErrorMessage: "",
	}
	data, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
	return http.StatusOK, nil
}

// AccountUpdateNameHandler updates a user's real name
func AccountUpdateNameHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)

	var auth updateAccount
	err := decoder.Decode(&auth)
	if err != nil {
		return http.StatusBadRequest, err
	}
	defer r.Body.Close()

	loggedIn, errorMessage, err := ctx.Database.SessionInformation(auth.SessionKey)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if !loggedIn {
		info := updateAccountStatus{
			Success:      false,
			ErrorMessage: errorMessage,
		}
		data, err := json.MarshalIndent(info, "", "\t")
		if err != nil {
			return http.StatusInternalServerError, err
		}

		// Send the JSON object back to the user
		w.Header().Add("content-type", "application/json")
		w.Write(data)
		return http.StatusOK, nil
	}

	errorMessage, err = ctx.Database.UpdateName(auth.SessionKey, auth.Name)

	status := updateAccountStatus{
		Success:      err == nil,
		ErrorMessage: errorMessage,
	}
	data, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
	return http.StatusOK, nil
}
