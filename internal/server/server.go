package server

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
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
	PublicKey string `json:"publicKey"`
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Generate a dummy public key for now (you can implement real key generation later)
	dummyPublicKey := "0000000000000000000000000000000000000000000000000000000000000000"

	err := s.userStorage.RegisterNewUser(req.Username, req.Password, dummyPublicKey)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Registration successful"})
	log.Printf("User registered: %s", req.Username)
}

// HandleLogin handles user login
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Verify user credentials
	isValid, err := s.userStorage.VerifyUser(req.Username, req.Password)
	if err != nil || !isValid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username or password"})
		return
	}

	// Generate a simple token (in production, use JWT or similar)
	token := "token_" + req.Username

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token":    token,
		"username": req.Username,
	})
	log.Printf("User logged in: %s", req.Username)
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
	json.NewEncoder(w).Encode(map[string]string{
		"username":  username,
		"publicKey": hex.EncodeToString(publicKey),
	})
}

// HandleConnections handles incoming websocket connections
func (s *Server) HandleConnections(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
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

// HandleRooms handles room-related requests (GET for list, POST for create)
func (s *Server) HandleRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method == "GET" {
		// Return empty rooms list for now
		json.NewEncoder(w).Encode([]map[string]interface{}{})
		return
	}
	
	if r.Method == "POST" {
		// TODO: Implement room creation
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Room created"})
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleMessages handles message-related requests
func (s *Server) HandleMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Return empty messages list for now
	json.NewEncoder(w).Encode([]map[string]interface{}{})
}

func Start() {
	server := NewServer()

	// API endpoints for frontend
	http.HandleFunc("/api/register", server.HandleRegister)
	http.HandleFunc("/api/login", server.HandleLogin)
	http.HandleFunc("/api/rooms", server.HandleRooms)
	http.HandleFunc("/api/messages/", server.HandleMessages)
	
	// Original endpoints (for backward compatibility)
	http.HandleFunc("/register", server.HandleRegister)
	http.HandleFunc("/keys/", server.HandleGetPublicKey)
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