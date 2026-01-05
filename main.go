package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.etcd.io/bbolt"
)

//go:embed web/*
var webFiles embed.FS

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // In production, implement proper origin checking
		},
	}
	db *bbolt.DB
)

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from any origin (for development)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Initialize database
	var err error
	db, err = bbolt.Open("superchat.db", 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize database buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{"users", "teams", "messages", "team_members", "uploads", "upload_data"}
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Start TTL janitors (7 days TTL, run hourly)
	startMessageTTLJanitor(7*24*time.Hour, 1*time.Hour)
	startUploadTTLJanitor(7*24*time.Hour, 1*time.Hour)

	// Create router
	r := mux.NewRouter()

	// Apply CORS middleware to all routes
	// TODO remove cors
	r.Use(corsMiddleware)

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Auth routes
	api.HandleFunc("/auth/register", handleRegister).Methods("POST")
	api.HandleFunc("/auth/login", handleLogin).Methods("POST")
	api.HandleFunc("/auth/logout", authMiddleware(handleLogout)).Methods("POST")

	// User routes
	api.HandleFunc("/user/profile", authMiddleware(handleGetProfile)).Methods("GET")
	api.HandleFunc("/user/profile", authMiddleware(handleUpdateProfile)).Methods("PUT")
	api.HandleFunc("/user/password", authMiddleware(handleChangePassword)).Methods("PUT")

	// Team routes (place static paths before variable {id} routes)
	api.HandleFunc("/teams", authMiddleware(handleCreateTeam)).Methods("POST")
	api.HandleFunc("/teams", authMiddleware(handleGetTeams)).Methods("GET")
	api.HandleFunc("/teams/online", authMiddleware(handleGetOnlineCounts)).Methods("GET")
	api.HandleFunc("/teams/{id}", authMiddleware(handleGetTeam)).Methods("GET")
	api.HandleFunc("/teams/{id}", authMiddleware(handleUpdateTeam)).Methods("PUT")
	api.HandleFunc("/teams/{id}", authMiddleware(handleDeleteTeam)).Methods("DELETE")
	api.HandleFunc("/teams/{id}/members", authMiddleware(handleGetTeamMembers)).Methods("GET")
	api.HandleFunc("/teams/{id}/members", authMiddleware(handleAddTeamMember)).Methods("POST")
	api.HandleFunc("/teams/{id}/members/{userId}", authMiddleware(handleRemoveTeamMember)).Methods("DELETE")
	api.HandleFunc("/teams/{id}/join", authMiddleware(handleJoinTeam)).Methods("POST")
	api.HandleFunc("/teams/{id}/leave", authMiddleware(handleLeaveTeam)).Methods("POST")

	// Chat routes
	api.HandleFunc("/teams/{id}/messages", authMiddleware(handleGetMessages)).Methods("GET")
	api.HandleFunc("/teams/{id}/messages", authMiddleware(handleSendMessage)).Methods("POST")
	api.HandleFunc("/teams/{id}/messages/{messageId}", authMiddleware(handleDeleteMessage)).Methods("DELETE")

	// Users
	api.HandleFunc("/users", authMiddleware(handleListUsers)).Methods("GET")
	// Uploads
	api.HandleFunc("/upload", authMiddleware(handleUpload)).Methods("POST")
	// GET for uploads uses token query param (not Authorization header) so it must not use authMiddleware
	api.HandleFunc("/uploads/{id}", handleGetUpload).Methods("GET")

	// WebSocket endpoint
	api.HandleFunc("/ws/{teamId}", handleWebSocket)

	// Redirect bare chat path (no team) to login to avoid client-side redirect loops
	r.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	}).Methods("GET")
	r.HandleFunc("/chat/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	}).Methods("GET")

	// Serve static files (SPA) from embedded /web at root
	staticContent, _ := fs.Sub(webFiles, "web")
	fsHandler := http.FileServer(http.FS(staticContent))

	// Serve assets directly (e.g., /assets/*)
	r.PathPrefix("/assets/").Handler(fsHandler)

	// SPA fallback: serve index.html for non-API routes
	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only handle non-API, non-WS routes here
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/ws/") || r.URL.Path == "/ws" {
			http.NotFound(w, r)
			return
		}

		// Try to serve the requested file if it exists
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		if f, err := staticContent.Open(strings.TrimPrefix(path, "/")); err == nil {
			f.Close()
			fsHandler.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for client-side routing
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/index.html"
		fsHandler.ServeHTTP(w, r2)
	}))

	host := flag.String("host", "", "Hostname or IP to bind the server (optional, defaults to local LAN IP)")
	port := flag.Int("port", 8443, "Port to run the server on")
	flag.Parse()

	bindHost := *host
	addr := fmt.Sprintf("%s:%d", bindHost, *port)
	fmt.Printf("Server starting on http://%s:%d/\n", bindHost, *port)
	log.Fatal(http.ListenAndServe(addr, r))
}
