package auth

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("meadowlark-secret-key-change-in-production") // Change in production!

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
	// Make public_key optional for now (can be NULL)
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		"username" TEXT NOT NULL PRIMARY KEY,
		"hashed_password" BLOB NOT NULL,
		"public_key" BLOB);`

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	return &UserStorage{db: db}
}

// RegisterNewUser creates a new user, hashes their password and stores them in the db
// publicKeyHex is optional - if empty, public_key will be NULL
func (s *UserStorage) RegisterNewUser(username, password string, publicKeyHex string) error {
	if username == "" || password == "" {
		return errors.New("username and password cannot be empty")
	}

	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var publicKeyBytes interface{}
	if publicKeyHex != "" {
		decoded, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			return errors.New("invalid public key format")
		}
		publicKeyBytes = decoded
	} else {
		publicKeyBytes = nil
	}

	insertSQL := `INSERT INTO users (username, hashed_password, public_key) VALUES (?, ?, ?)`
	_, err = s.db.Exec(insertSQL, username, hashedPassword, publicKeyBytes)
	if err != nil {
		return errors.New("username already exists")
	}

	return nil
}

// VerifyUser checks username and password, returns true if valid
func (s *UserStorage) VerifyUser(username, password string) error {
	querySQL := `SELECT hashed_password FROM users WHERE username = ?`
	var hashedPassword []byte

	err := s.db.QueryRow(querySQL, username).Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("invalid username or password")
		}
		return err
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		return errors.New("invalid username or password")
	}

	return nil
}

// UserClaims represents JWT claims
type UserClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(username string) (string, error) {
	claims := UserClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates a JWT token and returns the username
func ValidateToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims.Username, nil
	}

	return "", errors.New("invalid token")
}

// GetAllUsers returns a list of all registered usernames
func (s *UserStorage) GetAllUsers() ([]string, error) {
	querySQL := `SELECT username FROM users ORDER BY username`
	rows, err := s.db.Query(querySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		users = append(users, username)
	}

	return users, rows.Err()
}

// GetUserPublicKey retrieves a user's public key (returns error if no key is set)
func (s *UserStorage) GetUserPublicKey(username string) ([]byte, error) {
	// First check if user exists
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`, username).Scan(&exists)
	if err != nil || !exists {
		return nil, errors.New("user not found")
	}

	querySQL := `SELECT public_key FROM users WHERE username = ?`
	var publicKeyBytes []byte

	err = s.db.QueryRow(querySQL, username).Scan(&publicKeyBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		// NULL values in SQLite with go-sqlite3 driver will result in empty slice or scan error
		return nil, errors.New("user has no public key")
	}

	if len(publicKeyBytes) == 0 {
		return nil, errors.New("user has no public key")
	}
	return publicKeyBytes, nil
}
