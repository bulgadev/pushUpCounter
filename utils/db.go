package utils

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

var ErrUserNotFound = errors.New("user not found")

func OpenDuckDB(path string) (*sql.DB, error) {
	return sql.Open("duckdb", path)
}

func EnsureUsersTableSchema(db *sql.DB) error {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS users (name TEXT, psswd TEXT, count INTEGER)"); err != nil {
		return err
	}

	rows, err := db.Query("SELECT * FROM users LIMIT 0")
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	rows.Close()
	if err != nil {
		return err
	}

	columnSet := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		columnSet[strings.ToLower(column)] = struct{}{}
	}

	if _, ok := columnSet["token"]; !ok {
		if _, err := db.Exec("ALTER TABLE users ADD COLUMN token TEXT"); err != nil {
			return err
		}
	}

	if _, ok := columnSet["token_created_at"]; !ok {
		if _, err := db.Exec("ALTER TABLE users ADD COLUMN token_created_at TIMESTAMP"); err != nil {
			return err
		}
	}

	return nil
}

func AuthenticateUserAndGetToken(db *sql.DB, username, password string) (token string, createdAt time.Time, valid bool, err error) {
	var storedHash string
	var existingToken sql.NullString
	var tokenCreatedAt sql.NullTime

	err = db.QueryRow("SELECT psswd, token, token_created_at FROM users WHERE name = ? LIMIT 1", username).
		Scan(&storedHash, &existingToken, &tokenCreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, err
	}

	if !CheckPasswordHash(password, storedHash) {
		return "", time.Time{}, false, nil
	}

	if existingToken.Valid && strings.TrimSpace(existingToken.String) != "" {
		if tokenCreatedAt.Valid {
			return existingToken.String, tokenCreatedAt.Time.UTC(), true, nil
		}

		createdAt = time.Now().UTC()
		if _, err := db.Exec("UPDATE users SET token_created_at = ? WHERE name = ?", createdAt, username); err != nil {
			return "", time.Time{}, false, err
		}

		return existingToken.String, createdAt, true, nil
	}

	token, err = GenerateSessionToken()
	if err != nil {
		return "", time.Time{}, false, err
	}

	createdAt = time.Now().UTC()
	if _, err := db.Exec("UPDATE users SET token = ?, token_created_at = ? WHERE name = ?", token, createdAt, username); err != nil {
		return "", time.Time{}, false, err
	}

	return token, createdAt, true, nil
}

func FindUsernameByToken(db *sql.DB, token string) (string, error) {
	var username string
	err := db.QueryRow("SELECT name FROM users WHERE token = ? LIMIT 1", token).Scan(&username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return username, nil
}

func GetCountsAndTotals(db *sql.DB, totalGoal int) ([]int, int, int, error) {
	rows, err := db.Query("SELECT count FROM users")
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	var counts []int
	var totalCount int
	for rows.Next() {
		var count int
		if err := rows.Scan(&count); err != nil {
			return nil, 0, 0, err
		}
		counts = append(counts, count)
		totalCount += count
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, err
	}

	toGo := totalGoal - totalCount
	return counts, toGo, totalCount, nil
}

func RegisterPushUps(db *sql.DB, username string, pushUps int) error {
	if pushUps <= 0 {
		return fmt.Errorf("pushUps must be a positive number")
	}

	var existing int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE name = ?", username).Scan(&existing); err != nil {
		return err
	}
	if existing == 0 {
		return ErrUserNotFound
	}

	_, err := db.Exec("UPDATE users SET count = count + ? WHERE name = ?", pushUps, username)
	return err
}

func UpsertChallengeUser(db *sql.DB, username, plainPassword string) error {
	hashedPassword, err := HashPassword(plainPassword)
	if err != nil {
		return err
	}

	var existing int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE name = ?", username).Scan(&existing); err != nil {
		return err
	}

	if existing == 0 {
		_, err = db.Exec("INSERT INTO users(name, psswd, count) VALUES(?, ?, 0)", username, hashedPassword)
		return err
	}

	_, err = db.Exec("UPDATE users SET psswd = ?, token = NULL, token_created_at = NULL WHERE name = ?", hashedPassword, username)
	return err
}
