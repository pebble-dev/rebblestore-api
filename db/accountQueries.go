package db

import (
	"database/sql"
	"errors"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
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

func createSession(tx *sql.Tx, userId int) (string, error) {
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

	_, err = tx.Exec("INSERT INTO userSessions(sessionKey, userId, loginTime, lastSeenTime) VALUES (?, ?, ?, ?)", sessionKey, userId, time.Now().UnixNano(), time.Now().UnixNano())
	if err != nil {
		return "", err
	}

	return sessionKey, nil
}

// AccountRegister attempts to create a user account, and returns a user friendly error as well as an actual error (as not to display SQL statements to the user for example).
func (handler Handler) AccountRegister(username string, password string, realName string, remoteIp string) (string, string, error) {
	tx, err := handler.DB.Begin()
	if err != nil {
		return "", "Internal server error", err
	}
	defer tx.Rollback()

	// Check if user exists
	rows, err := tx.Query("SELECT username FROM users WHERE username=?", username)
	if err != nil {
		return "", "Internal server error", err
	}
	if rows.Next() {
		return "", "This username is already taken", errors.New("Username already taken")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "Internal server error", err
	}

	// Create user
	res, err := tx.Exec("INSERT INTO users(username, passwordHash, realName, disabled) VALUES (?, ?, ?, 0)", username, passwordHash, realName)
	if err != nil {
		return "", "Internal server error", err
	}
	userId, err := res.LastInsertId()
	if err != nil {
		return "", "Internal server error", err
	}

	// Create user session

	sessionKey, err := createSession(tx, int(userId))
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
func (handler Handler) AccountExists(username string) (bool, error) {
	var userId int
	row := handler.DB.QueryRow("SELECT id FROM users WHERE username=?", username)
	err := row.Scan(&userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// AccountRateLimited checks if an account is rate limited (too many recently failed logins)
func (handler Handler) AccountRateLimited(username string, remoteIp string) (bool, error) {
	var userId int
	row := handler.DB.QueryRow("SELECT id FROM users WHERE username=?", username)
	err := row.Scan(&userId)
	if err != nil {
		return false, err
	}

	// Get all login attempts for user in last hour
	var countUser int
	row = handler.DB.QueryRow("SELECT count(*) FROM userLogins WHERE userId=? AND time >= ? - 3600000000000", userId, time.Now().UnixNano())
	err = row.Scan(&countUser)
	if err != nil {
		return false, err
	}

	// Get all login attempts for client in last hour
	var countClient int
	row = handler.DB.QueryRow("SELECT count(*) FROM userLogins WHERE remoteIp=? AND time >= ? - 3600000000000", remoteIp, time.Now().UnixNano())
	err = row.Scan(&countClient)
	if err != nil {
		return false, err
	}

	// Rate limited if more than 10 login attemps
	return countUser > 10 || countClient > 10, nil
}

// AccountLogin returns a new session key, as well as a user-friendly error and an actual error
func (handler Handler) AccountLogin(username string, password string, remoteIp string) (string, string, error) {
	tx, err := handler.DB.Begin()
	if err != nil {
		return "", "Internal server error", err
	}
	defer tx.Rollback()

	var userId int
	var passwordHash string
	var disabled bool
	row := handler.DB.QueryRow("SELECT id, passwordHash, disabled FROM users WHERE username=?", username)
	err = row.Scan(&userId, &passwordHash, &disabled)
	if err != nil {
		return "", "Internal server error", err
	}

	if disabled {
		return "", "Account is disabled", errors.New("cannot login; account is disabled")
	}

	success := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil

	tx.Exec("INSERT INTO userLogins(userId, remoteIp, time, success) VALUES (?, ?, ?, ?)", userId, remoteIp, time.Now().UnixNano(), success)

	if success {
		sessionKey, err := createSession(tx, userId)
		tx.Commit()

		if err != nil {
			return "", "Internal server error", err
		}

		return sessionKey, "", nil
	}

	tx.Commit()

	return "", "Invalid password", errors.New("invalid password")
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
func (handler Handler) AccountInformation(sessionKey string) (bool, string, string, error) {
	userId, err := handler.getAccountId(sessionKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", "", nil
		}

		return false, "", "", err
	}

	var username, realName string
	row := handler.DB.QueryRow("SELECT username, realName FROM users WHERE id=?", userId)
	err = row.Scan(&username, &realName)
	if err != nil {
		return false, "", "", err
	}

	return true, username, realName, nil
}

// UpdatePassword updates a user's password and returns a human-readable error as well as an actual error
func (handler Handler) UpdatePassword(sessionKey string, password string) (string, error) {
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

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "Internal server error", err
	}

	tx.Exec("UPDATE users SET passwordHash=? WHERE id=?", hash, userId)
	if err != nil {
		return "Internal server error", err
	}

	err = tx.Commit()
	if err != nil {
		return "Internal server error", err
	}

	return "", nil
}

// UpdateRealName updates a user's real name and returns a human-readable error as well as an actual error
func (handler Handler) UpdateRealName(sessionKey string, realName string) (string, error) {
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

	tx.Exec("UPDATE users SET realName=? WHERE id=?", realName, userId)
	if err != nil {
		return "Internal server error", err
	}

	err = tx.Commit()
	if err != nil {
		return "Internal server error", err
	}

	return "", nil
}
