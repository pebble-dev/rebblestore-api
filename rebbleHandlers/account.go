package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
)

type accountInfo struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	RealName        string `json:"realName"`
	CaptchaResponse string `json:"captchaResponse"`
}

type loginInfo struct {
	SessionKey string `json:"sessionKey"`
}

type updateAccountInfo struct {
	SessionKey string `json:"sessionKey"`
	Password   string `json:"password"`
	RealName   string `json:"realName"`
}

type captchaStatus struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

type accountRegisterStatus struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
	SessionKey   string `json:"sessionKey"`
}

type accountLoginStatus struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
	SessionKey   string `json:"sessionKey"`
	RateLimited  bool   `json:"rateLimited"`
}

type accountLoggedInStatus struct {
	LoggedIn bool   `json:"loggedIn"`
	Username string `json:"username"`
	RealName string `json:"realName"`
}

type updateAccountStatus struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
}

func accountRegisterFail(message string, err error, w *http.ResponseWriter) error {
	status := accountRegisterStatus{
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

func accountLoginFail(message string, rateLimited bool, err error, w *http.ResponseWriter) error {
	status := accountLoginStatus{
		Success:      false,
		ErrorMessage: message,
		RateLimited:  rateLimited,
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

func checkCaptcha(ctx *HandlerContext, captchaResponse string, remoteAddr string) bool {
	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{"secret": {ctx.CaptchaSecret}, "response": {captchaResponse}, "remoteip": {remoteAddr}})
	if err != nil {
		return false
	}

	var captchaStatus captchaStatus
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&captchaStatus)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return captchaStatus.Success
}

// AccountRegisterHandler handles the creation of a user account
func AccountRegisterHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)
	var info accountInfo
	err := decoder.Decode(&info)
	if err != nil {
		return http.StatusBadRequest, err
	}
	defer r.Body.Close()

	captchaSuccess := checkCaptcha(ctx, info.CaptchaResponse, r.RemoteAddr)
	if !captchaSuccess {
		return http.StatusBadRequest, accountRegisterFail("Invalid CAPTCHA", errors.New("Invalid CAPTCHA"), &w)
	}

	// Password strength checking is done user-side with zxcvbn. If they decide, for whatever reason, to bypass that, they are only harming themselves.
	if len(info.Password) > 255 || len(info.Password) == 0 {
		return http.StatusBadRequest, accountRegisterFail("Invalid password", errors.New("Invalid password"), &w)
	}

	// Account creation
	sessionKey, userErr, err := ctx.Database.AccountRegister(info.Username, info.Password, info.RealName, r.RemoteAddr)
	if err != nil {
		return http.StatusBadRequest, accountRegisterFail(userErr, err, &w)
	}

	status := accountRegisterStatus{
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

// AccountLoginHandler handles the login of a user
func AccountLoginHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)

	// We also use accountInfo. Since this is just a login and not a register, not all fields require to be set (only username and password).
	// However, if the user is rate-limited, the user will need to complete a CAPTCHA. In this case, this field will also need to be checked.
	var info accountInfo
	err := decoder.Decode(&info)
	if err != nil {
		return http.StatusBadRequest, err
	}
	defer r.Body.Close()

	accountExists, err := ctx.Database.AccountExists(info.Username)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if info.Username == "" || len(info.Username) <= 3 || !accountExists {
		return http.StatusBadRequest, accountLoginFail("Invalid username", false, errors.New("Invalid username"), &w)
	}

	// Check if user is rate-limited
	rateLimited, err := ctx.Database.AccountRateLimited(info.Username, r.RemoteAddr)
	if rateLimited {
		captchaSuccess := checkCaptcha(ctx, info.CaptchaResponse, r.RemoteAddr)
		if !captchaSuccess {
			return http.StatusBadRequest, accountLoginFail("Invalid CAPTCHA", true, errors.New("Invalid CAPTCHA"), &w)
		}
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	sessionKey, userErr, err := ctx.Database.AccountLogin(info.Username, info.Password, r.RemoteAddr)
	if err != nil {
		return http.StatusBadRequest, accountLoginFail(userErr, rateLimited, err, &w)
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

// AccountStatusHandler displays the status for a given Session Key
func AccountStatusHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)

	// We also use accountInfo. Since this is just a login and not a register, not all fields require to be set (only username and password).
	// However, if the user is rate-limited, the user will need to complete a CAPTCHA. In this case, this field will also need to be checked.
	var info loginInfo
	err := decoder.Decode(&info)
	if err != nil {
		return http.StatusBadRequest, err
	}
	defer r.Body.Close()

	loggedIn, username, realName, err := ctx.Database.AccountInformation(info.SessionKey)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	status := accountLoggedInStatus{
		LoggedIn: loggedIn,
		Username: username,
		RealName: realName,
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

// AccountUpdatePasswordHandler updates a user's password
func AccountUpdatePasswordHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)

	var info updateAccountInfo
	err := decoder.Decode(&info)
	if err != nil {
		return http.StatusBadRequest, err
	}
	defer r.Body.Close()

	// Password strength checking is done user-side with zxcvbn. If they decide, for whatever reason, to bypass that, they are only harming themselves.
	if len(info.Password) > 255 || len(info.Password) == 0 {
		return http.StatusBadRequest, accountRegisterFail("Invalid password", errors.New("Invalid password"), &w)
	}

	errorMessage, err := ctx.Database.UpdatePassword(info.SessionKey, info.Password)

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

// AccountUpdateRealNameHandler updates a user's real name
func AccountUpdateRealNameHandler(ctx *HandlerContext, w http.ResponseWriter, r *http.Request) (int, error) {
	decoder := json.NewDecoder(r.Body)

	var info updateAccountInfo
	err := decoder.Decode(&info)
	if err != nil {
		return http.StatusBadRequest, err
	}
	defer r.Body.Close()

	errorMessage, err := ctx.Database.UpdateRealName(info.SessionKey, info.RealName)

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
