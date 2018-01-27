package db

import (
	"database/sql"
	"errors"
	"math/rand"
	"time"
)

func init() {
	// We need to seed the RNG which is used by generateSessionId()
	rand.Seed(time.Now().UnixNano())
}

// generateSessionKey() generates a pseudo-random fixed-length base64 string
func generateSessionKey() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_"

	b := make([]byte, 50)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func createSession(tx *sql.Tx, userId int, accessToken string, expires int64) (string, error) {
	sessionKey := generateSessionKey()

	// We check that we have no more than 5 active sessions at a time (for security reasons). Otherwise, we delete the oldest one.
	var count int
	row := tx.QueryRow("SELECT count(*) FROM userSessions WHERE userId=?", userId)
	err := row.Scan(&count)
	if err != nil {
		return "", err
	}

	if count >= 5 {
		_, err = tx.Exec("DELETE FROM userSessions WHERE sessionKey=(SELECT sessionKey FROM userSessions WHERE userId=? ORDER BY lastSeenTime ASC LIMIT 1)", userId)
		if err != nil {
			return "", err
		}
	}

	_, err = tx.Exec("INSERT INTO userSessions(sessionKey, userId, access_token, expires) VALUES (?, ?, ?, ?)", sessionKey, userId, accessToken, expires)
	if err != nil {
		return "", err
	}

	return sessionKey, nil
}

// AccountLoginOrRegister attempts to login (or, if the user doesn't yet exist, create a user account), and returns a user friendly error as well as an actual error (as not to display SQL statements to the user for example).
func (handler Handler) AccountLoginOrRegister(provider string, sub string, name string, accessToken string, expires int64, remoteIp string) (string, string, error) {
	tx, err := handler.DB.Begin()
	if err != nil {
		return "", "Internal server error", err
	}
	defer tx.Rollback()

	row := tx.QueryRow("SELECT id, disabled FROM users WHERE provider=? AND sub=?", provider, sub)

	var userId int64
	disabled := false
	err = row.Scan(&userId, &disabled)
	if err != nil {
		// User doesn't exist, create account

		// Create user
		res, err := tx.Exec("INSERT INTO users(provider, sub, name, pebbleMirror, disabled) VALUES (?, ?, ?, 0, 0)", provider, sub, name)
		if err != nil {
			return "", "Internal server error", err
		}
		userId, err = res.LastInsertId()
		if err != nil {
			return "", "Internal server error", err
		}
	}

	if disabled {
		return "", "Account is disabled", errors.New("cannot login; account is disabled")
	}

	// Create user session

	sessionKey, err := createSession(tx, int(userId), accessToken, expires)
	if err != nil {
		return "", "Internal server error", err
	}

	// Log successful login attempt
	_, err = tx.Exec("INSERT INTO userLogins(userId, remoteIp, time, success) VALUES (?, ?, ?, 1)", userId, remoteIp, time.Now().UnixNano())
	if err != nil {
		return "", "Internal server error", err
	}

	tx.Commit()

	return sessionKey, "", nil
}

// AccountExists checks if an account exists
func (handler Handler) AccountExists(provider string, sub string) (bool, error) {
	var userId int
	row := handler.DB.QueryRow("SELECT id FROM users WHERE provider=? AND sub=?", provider, sub)
	err := row.Scan(&userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (handler Handler) getAccountId(sessionKey string) (int, error) {
	var userId int
	row := handler.DB.QueryRow("SELECT userId FROM userSessions WHERE sessionKey=?", sessionKey)
	err := row.Scan(&userId)
	if err != nil {
		return 0, err
	}

	return userId, nil
}

// AccountInformation returns information about the account associated to the given session key
func (handler Handler) AccountInformation(sessionKey string) (bool, string, error) {
	userId, err := handler.getAccountId(sessionKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}

		return false, "", err
	}

	var name, provider, sub string
	row := handler.DB.QueryRow("SELECT name, provider, sub FROM users WHERE id=?", userId)
	err = row.Scan(&name, &provider, &sub)
	if err != nil {
		return false, "", err
	}

	if name == "" {
		return true, provider + "_" + sub, nil
	}

	return true, name, nil
}

// UpdateName updates a user's name and returns a human-readable error as well as an actual error
func (handler Handler) UpdateName(sessionKey string, name string) (string, error) {
	userId, err := handler.getAccountId(sessionKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Invalid session key", errors.New("Invalid session key")
		}

		return "Internal server error", err
	}

	tx, err := handler.DB.Begin()
	if err != nil {
		return "Internal server error", err
	}
	defer tx.Rollback()

	tx.Exec("UPDATE users SET name=? WHERE id=?", name, userId)
	if err != nil {
		return "Internal server error", err
	}

	err = tx.Commit()
	if err != nil {
		return "Internal server error", err
	}

	return "", nil
}
