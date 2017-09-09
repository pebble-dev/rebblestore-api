package rebbleHandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"golang.org/x/crypto/bcrypt"
)

type accountInfo struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	RealName        string `json:"realName"`
	CaptchaResponse string `json:"captchaResponse"`
}

type captchaStatus struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

type accountCreationStatus struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
}

func accountCreationFail(message string, err error, w *http.ResponseWriter) error {
	status := accountCreationStatus{
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

	return err
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

	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{"secret": {ctx.CaptchaSecret}, "response": {info.CaptchaResponse}, "remoteip": {r.RemoteAddr}})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var captchaStatus captchaStatus
	decoder = json.NewDecoder(resp.Body)
	err = decoder.Decode(&captchaStatus)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	if info.Username == "" || len(info.Username) <= 3 {
		return http.StatusBadRequest, accountCreationFail("Invalid username", errors.New("Invalid username"), &w)
	}

	if !captchaStatus.Success {
		return http.StatusBadRequest, accountCreationFail("Invalid CAPTCHA", errors.New("Invalid CAPTCHA"), &w)
	}

	// Password strength checking is done user-side with zxcvbn. If they decide, for whatever reason, to bypass that, they are only harming themselves.
	if len(info.Password) > 255 || len(info.Password) == 0 {
		return http.StatusBadRequest, accountCreationFail("Invalid password", errors.New("Invalid password"), &w)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(info.Password), bcrypt.DefaultCost)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	userErr, err := ctx.Database.AddAccount(info.Username, passwordHash, info.RealName)
	if err != nil {
		return http.StatusBadRequest, accountCreationFail(userErr, err, &w)
	}

	status := accountCreationStatus{
		Success:      true,
		ErrorMessage: userErr,
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
