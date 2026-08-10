package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"my-ticket/db"
	"my-ticket/handlers"
)

func main() {
	dbPath := "./tickets.db"
	log.Printf("Connecting to SQLite database at %s...", dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	h := handlers.NewHandler(database)

	mux := http.NewServeMux()

	// Serve Static Files
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Auth Routes (Public)
	mux.HandleFunc("GET /login", h.HandleLoginForm)
	mux.HandleFunc("POST /login", h.HandleLogin)
	mux.HandleFunc("POST /logout", h.HandleLogout)

	// Public Routes (Open to Anonymous & All Visitors)
	mux.HandleFunc("GET /", h.HandleIndex)
	mux.HandleFunc("GET /tickets", h.HandleTickets)
	mux.HandleFunc("GET /tickets/stats", h.HandleStats)
	mux.HandleFunc("GET /tickets/new", h.HandleNewTicketForm)
	mux.HandleFunc("POST /tickets", h.HandleCreateTicket) // Anonymous creation allowed!
	mux.HandleFunc("GET /tickets/{id}", h.HandleTicketDetail)
	mux.HandleFunc("POST /tickets/{id}/comments", h.HandleAddComment)

	// Protected Private Routes (Require Admin JWT Permission)
	mux.HandleFunc("GET /tickets/{id}/edit", h.RequireAdmin(h.HandleEditTicketForm))
	mux.HandleFunc("POST /tickets/{id}/edit", h.RequireAdmin(h.HandleUpdateTicket))
	mux.HandleFunc("POST /tickets/{id}/status", h.RequireAdmin(h.HandleUpdateStatus))
	mux.HandleFunc("DELETE /tickets/{id}", h.RequireAdmin(h.HandleDeleteTicket))

	// Wrap entire mux with JWTMiddleware to populate user context
	handlerWithJWT := h.JWTMiddleware(mux)

	port := os.Getenv("PORT")
	if _, err := strconv.Atoi(port); err != nil {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("==================================================")
	//nolint:gosec // G706: port is validated via strconv.Atoi
	log.Printf("🚀 DevTicket Pro server starting on http://localhost:%s", port)
	log.Printf("   Admin credentials: username='admin', password='admin123'")
	log.Printf("==================================================")

	server := &http.Server{
		Addr:              addr,
		Handler:           handlerWithJWT,
		ReadHeaderTimeout: 3 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
