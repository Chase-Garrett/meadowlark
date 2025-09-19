package auth

import (
	"database/sql"
	"encoding/hex"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"log"
)

// UserStorage manages user accounts in SQLite
type UserStorage struct {
	db *sql.DB
}

// NewUserStorage connects to SQLite and initalizes the users table
func NewUserStorage(dbPath string) *UserStorage {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create the users table if it doesn't already exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		"username" TEXT NOT NULL PRIMARY KEY,
		"hashed_password" BLOB NOT NULL,
		"public_key" BLOB NOT NULL);`

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	return &UserStorage{db: db}
}

// RegisterNewUser creates a new user, hashes their password and stores them in the db
func (s *UserStorage) RegisterNewUser(username, password string, publicKeyHex string) error {
	if username == "" || password == "" {
		return errors.New("username and password cannot be empty")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return errors.New("invalid public key format")
	}

	insertSQL := `INSERT INTO users (username, hashed_password, public_key) VALUES (?, ?, ?)`
	_, err = s.db.Exec(insertSQL, username, hashedPassword, publicKeyBytes)
	if err != nil {
		return errors.New("username already exists")
	}

	return nil
}

// GetUserPublicKey retrieves a user's public key
func (s *UserStorage) GetPublicKey(username string) ([]byte, error) {
	querySQL := `SELECT public_key FROM users WHERE username = ?`
	var publicKey []byte

	err := s.db.QueryRow(querySQL, username).Scan(&publicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return publicKey, nil
}
