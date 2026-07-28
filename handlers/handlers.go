package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"my-ticket/models"
)

type Handler struct {
	DB        *sql.DB
	Templates *template.Template
}

func NewHandler(db *sql.DB) *Handler {
	funcMap := template.FuncMap{
		"statusClass":   models.StatusClass,
		"priorityClass": models.PriorityClass,
		"typeIcon":      models.TypeIcon,
		"add": func(a, b int) int {
			return a + b
		},
		"eq": func(a, b string) bool {
			return a == b
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		log.Fatalf("Error parsing templates: %v", err)
	}

	return &Handler{
		DB:        db,
		Templates: tmpl,
	}
}

// Middleware: Extract JWT claims and put into request context
func (h *Handler) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("jwt_token")
		if err == nil && cookie.Value != "" {
			claims, err := models.ValidateToken(cookie.Value)
			if err == nil {
				ctx := context.WithValue(r.Context(), models.UserContextKey, claims)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Middleware: Require Admin role
func (h *Handler) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := models.GetUserFromContext(r.Context())
		if claims == nil || claims.Role != "admin" {
			w.Header().Set("HX-Trigger", "unauthorized")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`
				<div class="fixed bottom-6 right-6 z-50 flex items-center gap-3 bg-rose-950/90 border border-rose-500/50 text-rose-300 px-4 py-3 rounded-xl shadow-2xl backdrop-blur-md transition-all animate-bounce" id="toast">
					<i class="fa-solid fa-lock text-rose-400 text-lg"></i>
					<div>
						<p class="font-medium text-sm">Access Denied (401)</p>
						<p class="text-xs text-rose-400/80">Admin permission required via JWT token. Please login as Admin.</p>
					</div>
				</div>
			`))
			return
		}
		next(w, r)
	}
}

func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	filter := models.TicketFilter{
		Query:     r.URL.Query().Get("q"),
		Status:    r.URL.Query().Get("status"),
		Priority:  r.URL.Query().Get("priority"),
		Type:      r.URL.Query().Get("type"),
		Component: r.URL.Query().Get("component"),
		ViewMode:  r.URL.Query().Get("view"),
	}

	if filter.ViewMode == "" {
		filter.ViewMode = "kanban"
	}

	tickets, err := models.GetTickets(h.DB, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stats, err := models.GetStats(h.DB)
	if err != nil {
		log.Printf("Error fetching stats: %v", err)
	}

	userClaims := models.GetUserFromContext(r.Context())

	data := struct {
		Tickets    []models.Ticket
		Stats      models.TicketStats
		Filter     models.TicketFilter
		Toast      string
		KanbanCols map[string][]models.Ticket
		User       *models.JWTClaims
		IsAdmin    bool
	}{
		Tickets:    tickets,
		Stats:      stats,
		Filter:     filter,
		KanbanCols: groupTicketsByStatus(tickets),
		User:       userClaims,
		IsAdmin:    userClaims != nil && userClaims.Role == "admin",
	}

	err = h.Templates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("Template render error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleTickets(w http.ResponseWriter, r *http.Request) {
	filter := models.TicketFilter{
		Query:     r.URL.Query().Get("q"),
		Status:    r.URL.Query().Get("status"),
		Priority:  r.URL.Query().Get("priority"),
		Type:      r.URL.Query().Get("type"),
		Component: r.URL.Query().Get("component"),
		ViewMode:  r.URL.Query().Get("view"),
	}

	if filter.ViewMode == "" {
		filter.ViewMode = "kanban"
	}

	tickets, err := models.GetTickets(h.DB, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userClaims := models.GetUserFromContext(r.Context())

	data := struct {
		Tickets    []models.Ticket
		Filter     models.TicketFilter
		KanbanCols map[string][]models.Ticket
		IsAdmin    bool
	}{
		Tickets:    tickets,
		Filter:     filter,
		KanbanCols: groupTicketsByStatus(tickets),
		IsAdmin:    userClaims != nil && userClaims.Role == "admin",
	}

	templateName := "kanban_board.html"
	if filter.ViewMode == "table" {
		templateName = "ticket_table.html"
	}

	err = h.Templates.ExecuteTemplate(w, templateName, data)
	if err != nil {
		log.Printf("Template error in HandleTickets: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleTicketDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	ticket, err := models.GetTicketByID(h.DB, id)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	userClaims := models.GetUserFromContext(r.Context())

	data := struct {
		*models.Ticket
		IsAdmin bool
	}{
		Ticket:  ticket,
		IsAdmin: userClaims != nil && userClaims.Role == "admin",
	}

	err = h.Templates.ExecuteTemplate(w, "ticket_detail_modal.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleNewTicketForm(w http.ResponseWriter, r *http.Request) {
	userClaims := models.GetUserFromContext(r.Context())
	data := struct {
		Ticket  *models.Ticket
		IsAdmin bool
	}{
		Ticket:  nil,
		IsAdmin: userClaims != nil && userClaims.Role == "admin",
	}

	err := h.Templates.ExecuteTemplate(w, "ticket_form_modal.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleEditTicketForm(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	ticket, err := models.GetTicketByID(h.DB, id)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	userClaims := models.GetUserFromContext(r.Context())

	data := struct {
		*models.Ticket
		IsAdmin bool
	}{
		Ticket:  ticket,
		IsAdmin: userClaims != nil && userClaims.Role == "admin",
	}

	err = h.Templates.ExecuteTemplate(w, "ticket_form_modal.html", data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Anonymous Ticket Creation (Open to everyone!)
func (h *Handler) HandleCreateTicket(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	userClaims := models.GetUserFromContext(r.Context())

	sp, _ := strconv.Atoi(r.FormValue("story_points"))
	if sp <= 0 {
		sp = 1
	}

	reporter := r.FormValue("reporter")
	if reporter == "" {
		if userClaims != nil {
			reporter = userClaims.Username
		} else {
			reporter = "Anonymous User"
		}
	}

	ticket := models.Ticket{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Type:        r.FormValue("type"),
		Priority:    r.FormValue("priority"),
		Status:      r.FormValue("status"),
		Assignee:    r.FormValue("assignee"),
		Reporter:    reporter,
		Component:   r.FormValue("component"),
		StoryPoints: sp,
	}

	if ticket.Title == "" {
		ticket.Title = "Untitled Ticket"
	}
	if ticket.Status == "" {
		ticket.Status = "Backlog"
	}
	if ticket.Priority == "" {
		ticket.Priority = "Medium"
	}
	if ticket.Type == "" {
		ticket.Type = "Task"
	}
	if ticket.Assignee == "" {
		ticket.Assignee = "Unassigned"
	}

	err := models.CreateTicket(h.DB, &ticket)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create ticket: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refresh-board, refresh-stats, close-modal")
	w.WriteHeader(http.StatusOK)

	reporterBadge := "Anonymous"
	if userClaims != nil {
		reporterBadge = userClaims.Username
	}

	w.Write([]byte(`
		<div class="fixed bottom-6 right-6 z-50 flex items-center gap-3 bg-emerald-950/90 border border-emerald-500/50 text-emerald-300 px-4 py-3 rounded-xl shadow-2xl backdrop-blur-md transition-all animate-bounce" id="toast">
			<i class="fa-solid fa-circle-check text-emerald-400 text-lg"></i>
			<div>
				<p class="font-medium text-sm">Ticket Created (` + reporterBadge + `)</p>
				<p class="text-xs text-emerald-400/80">` + ticket.TicketKey + `: ` + template.HTMLEscapeString(ticket.Title) + ` created successfully.</p>
			</div>
		</div>
	`))
}

func (h *Handler) HandleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	sp, _ := strconv.Atoi(r.FormValue("story_points"))
	if sp <= 0 {
		sp = 1
	}

	ticket := models.Ticket{
		ID:          id,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Type:        r.FormValue("type"),
		Priority:    r.FormValue("priority"),
		Status:      r.FormValue("status"),
		Assignee:    r.FormValue("assignee"),
		Reporter:    r.FormValue("reporter"),
		Component:   r.FormValue("component"),
		StoryPoints: sp,
	}

	err = models.UpdateTicket(h.DB, &ticket)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update ticket: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refresh-board, close-modal")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<div class="fixed bottom-6 right-6 z-50 flex items-center gap-3 bg-indigo-950/90 border border-indigo-500/50 text-indigo-300 px-4 py-3 rounded-xl shadow-2xl backdrop-blur-md transition-all animate-bounce" id="toast">
			<i class="fa-solid fa-pen-to-square text-indigo-400 text-lg"></i>
			<div>
				<p class="font-medium text-sm">Ticket Updated (Admin)</p>
				<p class="text-xs text-indigo-400/80">Changes saved successfully.</p>
			</div>
		</div>
	`))
}

func (h *Handler) HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")
	if status == "" {
		status = r.URL.Query().Get("status")
	}

	if status != "" {
		_ = models.UpdateTicketStatus(h.DB, id, status)
	}

	w.Header().Set("HX-Trigger", "refresh-board, refresh-stats")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleDeleteTicket(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = models.DeleteTicket(h.DB, id)
	if err != nil {
		http.Error(w, "Failed to delete ticket", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refresh-board, refresh-stats, close-modal")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<div class="fixed bottom-6 right-6 z-50 flex items-center gap-3 bg-rose-950/90 border border-rose-500/50 text-rose-300 px-4 py-3 rounded-xl shadow-2xl backdrop-blur-md transition-all" id="toast">
			<i class="fa-solid fa-trash-can text-rose-400 text-lg"></i>
			<div>
				<p class="font-medium text-sm">Ticket Deleted (Admin)</p>
				<p class="text-xs text-rose-400/80">Ticket #` + idStr + ` has been deleted.</p>
			</div>
		</div>
	`))
}

func (h *Handler) HandleAddComment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	userClaims := models.GetUserFromContext(r.Context())
	author := r.FormValue("author")
	if author == "" {
		if userClaims != nil {
			author = userClaims.Username + " (Admin)"
		} else {
			author = "Anonymous Visitor"
		}
	}

	content := r.FormValue("content")
	if content != "" {
		comment := models.Comment{
			TicketID: id,
			Author:   author,
			Content:  content,
		}
		_ = models.AddComment(h.DB, &comment)
	}

	comments, err := models.GetCommentsByTicketID(h.DB, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.Templates.ExecuteTemplate(w, "comment_list.html", comments)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := models.GetStats(h.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.Templates.ExecuteTemplate(w, "stats_cards.html", stats)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Authentication Handlers

func (h *Handler) HandleLoginForm(w http.ResponseWriter, r *http.Request) {
	err := h.Templates.ExecuteTemplate(w, "login_modal.html", nil)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := models.AuthenticateUser(h.DB, username, password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`
			<div class="p-3 bg-rose-500/10 border border-rose-500/30 text-rose-400 rounded-xl text-xs flex items-center gap-2" id="login-error">
				<i class="fa-solid fa-circle-exclamation"></i>
				<span>Invalid username or password. Try <strong>admin</strong> / <strong>admin123</strong></span>
			</div>
		`))
		return
	}

	tokenStr, err := models.GenerateToken(user.Username, user.Role)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    tokenStr,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func groupTicketsByStatus(tickets []models.Ticket) map[string][]models.Ticket {
	cols := map[string][]models.Ticket{
		"Backlog":     {},
		"In Progress": {},
		"In Review":   {},
		"Done":        {},
	}
	for _, t := range tickets {
		cols[t.Status] = append(cols[t.Status], t)
	}
	return cols
}
