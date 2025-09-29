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
	Password  string `json:"password"`
	PublicKey string `json:"publicKey"`
}

// HandleRegister handles the registration of a user
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.userStorage.RegisterNewUser(req.Username, req.Password, req.PublicKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	log.Printf("User registered: %s", req.Username)
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

func Start() {
	server := NewServer()

	http.HandleFunc("/register", server.HandleRegister)
	http.HandleFunc("/keys/", server.HandleGetPublicKey)
	http.HandleFunc("/ws", server.HandleConnections)

	log.Println("HTTP server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
