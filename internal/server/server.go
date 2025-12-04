package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chase-Garrett/meadowlark/internal/auth"
	"github.com/Chase-Garrett/meadowlark/internal/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// server holds all dependencies for meadowlark application
type Server struct {
	userStorage *auth.UserStorage
	hub         *Hub
}

// create a new server instance
func NewServer() *Server {
	userStorage := auth.NewUserStorage("./chat.db")
	hub := NewHub()
	go hub.Run()
	return &Server{
		userStorage: userStorage,
		hub:         hub,
	}
}

// RegistrationRequest defines JSON for the /register endpoint
type RegistrationRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Email     string `json:"email"`     // Frontend sends this, we'll accept it but not store it yet
	PublicKey string `json:"publicKey"` // Optional
}

// LoginRequest defines JSON for the /api/login endpoint
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse defines JSON response for login
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// LoginRequest defines JSON for the /login endpoint
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleRegister handles the registration of a user
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Email is accepted but not stored yet (for future use)
	// PublicKey is optional
	err := s.userStorage.RegisterNewUser(req.Username, req.Password, req.PublicKey)
	if err != nil {
		respondJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Registration successful",
	})
	log.Printf("User registered: %s", req.Username)
}

// HandleLogin handles user login and returns JWT token
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.userStorage.VerifyUser(req.Username, req.Password)
	if err != nil {
		respondJSONError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(req.Username)
	if err != nil {
		respondJSONError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Token:    token,
		Username: req.Username,
	})
	log.Printf("User logged in: %s", req.Username)
}

// HandleGetUsers returns a list of all users (for direct messaging)
func (s *Server) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userStorage.GetAllUsers()
	if err != nil {
		respondJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Helper function to respond with JSON error
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Middleware to authenticate JWT tokens
func (s *Server) authenticateRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	username, err := auth.ValidateToken(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid token: %v", err)
	}

	return username, nil
}

// HandleGetPublicKey serves a user's publickey
func (s *Server) HandleGetPublicKey(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.URL.Path, "/keys/")
	publicKey, err := s.userStorage.GetUserPublicKey(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Return public key as base64 (Web Crypto API format)
	json.NewEncoder(w).Encode(map[string]string{
		"username":  username,
		"publicKey": base64.StdEncoding.EncodeToString(publicKey),
	})
}

// HandleConnections handles incoming websocket connections
func (s *Server) HandleConnections(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter or Authorization header
	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}
	}

	if token == "" {
		http.Error(w, "Authentication token required", http.StatusUnauthorized)
		return
	}

	username, err := auth.ValidateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: s.hub, conn: conn, send: make(chan *protocol.Message, 256), username: username}
	client.hub.register <- client

	log.Printf("Client connected: %s", username)

	go client.writePump()
	go client.readPump()
}

// ServeStaticFiles serves static files from the static directory
func (s *Server) ServeStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Skip API routes
	if strings.HasPrefix(r.URL.Path, "/api") ||
		strings.HasPrefix(r.URL.Path, "/ws") ||
		strings.HasPrefix(r.URL.Path, "/keys") ||
		strings.HasPrefix(r.URL.Path, "/register") {
		http.NotFound(w, r)
		return
	}

	// Get the requested path
	path := r.URL.Path
	if path == "/" || path == "" {
		path = "/index.html"
	}

	// Remove leading slash and build full path
	localPath := strings.TrimPrefix(path, "/")
	fullPath := filepath.Join("cmd", "static", localPath)

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist and it's not a root request, try index.html (for SPA routing)
			if path != "/index.html" {
				fullPath = filepath.Join("cmd", "static", "index.html")
				info, err = os.Stat(fullPath)
			}
			if err != nil {
				http.NotFound(w, r)
				return
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Prevent directory listing
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Set appropriate content type based on extension
	ext := filepath.Ext(fullPath)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, fullPath)
}

func Start() {
	server := NewServer()

	// Static file serving
	http.HandleFunc("/", server.ServeStaticFiles)

	// API endpoints
	http.HandleFunc("/api/register", server.HandleRegister)
	http.HandleFunc("/api/login", server.HandleLogin)
	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Authenticate the request
		_, err := server.authenticateRequest(r)
		if err != nil {
			respondJSONError(w, err.Error(), http.StatusUnauthorized)
			return
		}
		server.HandleGetUsers(w, r)
	})

	// Legacy endpoints (kept for compatibility)
	http.HandleFunc("/register", server.HandleRegister)
	http.HandleFunc("/keys/", server.HandleGetPublicKey)

	// WebSocket endpoint
	http.HandleFunc("/ws", server.HandleConnections)

	// Serve static files (must be last)
	fs := http.FileServer(http.Dir("./cmd/static"))
	http.Handle("/", fs)

	log.Println("HTTP server started on :8080")
	log.Println("Serving static files from: ./cmd/static")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}