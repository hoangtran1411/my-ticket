package handlers

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	mydb "my-ticket/db"
	"my-ticket/models"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// newTestHandler creates a Handler backed by an in-memory SQLite database (with
// seeded sample data) and templates parsed from the sibling templates/ dir.
func newTestHandler(t *testing.T) *Handler {
	t.Helper()

	database, err := mydb.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	funcMap := template.FuncMap{
		"statusClass":   models.StatusClass,
		"priorityClass": models.PriorityClass,
		"typeIcon":      models.TypeIcon,
		"add":           func(a, b int) int { return a + b },
		"eq":            func(a, b string) bool { return a == b },
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("../templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	return &Handler{DB: database, Templates: tmpl}
}

// adminCtx returns a context populated with admin JWT claims.
func adminCtx(r *http.Request) *http.Request {
	claims := &models.JWTClaims{Username: "admin", Role: "admin"}
	ctx := context.WithValue(r.Context(), models.UserContextKey, claims)
	return r.WithContext(ctx)
}

// firstTicketID returns the ID of the first ticket in the seeded test database.
func firstTicketID(t *testing.T, h *Handler) string {
	t.Helper()
	tickets, err := models.GetTickets(h.DB, models.TicketFilter{})
	if err != nil || len(tickets) == 0 {
		t.Fatalf("firstTicketID: no tickets in DB: %v", err)
	}
	return strings.TrimSpace(strings.Split(tickets[0].TicketKey, "-")[1])
}

func firstTicketNumID(t *testing.T, h *Handler) int64 {
	t.Helper()
	tickets, _ := models.GetTickets(h.DB, models.TicketFilter{})
	if len(tickets) == 0 {
		t.Fatal("no tickets in test DB")
	}
	return tickets[0].ID
}

// ─── Template parsing ─────────────────────────────────────────────────────────

func TestParseTemplates(t *testing.T) {
	funcMap := template.FuncMap{
		"statusClass":   models.StatusClass,
		"priorityClass": models.PriorityClass,
		"typeIcon":      models.TypeIcon,
		"add":           func(a, b int) int { return a + b },
		"eq":            func(a, b string) bool { return a == b },
	}
	_, err := template.New("").Funcs(funcMap).ParseGlob("../templates/*.html")
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}
}

// ─── groupTicketsByStatus ─────────────────────────────────────────────────────

func TestGroupTicketsByStatus(t *testing.T) {
	tickets := []models.Ticket{
		{ID: 1, Status: "Backlog"},
		{ID: 2, Status: "In Progress"},
		{ID: 3, Status: "In Progress"},
		{ID: 4, Status: "Done"},
	}
	cols := groupTicketsByStatus(tickets)

	if len(cols["Backlog"]) != 1 {
		t.Errorf("Backlog: got %d, want 1", len(cols["Backlog"]))
	}
	if len(cols["In Progress"]) != 2 {
		t.Errorf("In Progress: got %d, want 2", len(cols["In Progress"]))
	}
	if len(cols["Done"]) != 1 {
		t.Errorf("Done: got %d, want 1", len(cols["Done"]))
	}
	if len(cols["In Review"]) != 0 {
		t.Errorf("In Review: got %d, want 0", len(cols["In Review"]))
	}
}

func TestGroupTicketsByStatus_Empty(t *testing.T) {
	cols := groupTicketsByStatus(nil)
	for _, status := range []string{"Backlog", "In Progress", "In Review", "Done"} {
		if _, ok := cols[status]; !ok {
			t.Errorf("missing key %q in result", status)
		}
	}
}

// ─── Middleware: RequireAdmin ──────────────────────────────────────────────────

func TestRequireAdmin_NoAuth_Returns401(t *testing.T) {
	h := newTestHandler(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rr := httptest.NewRecorder()

	h.RequireAdmin(next)(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if rr.Header().Get("HX-Trigger") != "unauthorized" {
		t.Errorf("expected HX-Trigger=unauthorized, got %q", rr.Header().Get("HX-Trigger"))
	}
}

func TestRequireAdmin_AsAdmin_CallsNext(t *testing.T) {
	h := newTestHandler(t)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req = adminCtx(req)
	rr := httptest.NewRecorder()

	h.RequireAdmin(next)(rr, req)

	if !called {
		t.Error("expected next handler to be called for admin")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRequireAdmin_UserRole_Returns401(t *testing.T) {
	h := newTestHandler(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	claims := &models.JWTClaims{Username: "regular", Role: "user"}
	ctx := context.WithValue(context.Background(), models.UserContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	h.RequireAdmin(next)(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("non-admin user should get 401, got %d", rr.Code)
	}
}

// ─── Middleware: JWTMiddleware ────────────────────────────────────────────────

func TestJWTMiddleware_WithValidToken_SetsContext(t *testing.T) {
	h := newTestHandler(t)
	tokenStr, _ := models.GenerateToken("admin", "admin")

	var gotClaims *models.JWTClaims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = models.GetUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "jwt_token", Value: tokenStr})
	rr := httptest.NewRecorder()

	h.JWTMiddleware(next).ServeHTTP(rr, req)

	if gotClaims == nil {
		t.Fatal("expected claims in context, got nil")
	}
	if gotClaims.Username != "admin" {
		t.Errorf("Username: got %q, want %q", gotClaims.Username, "admin")
	}
}

func TestJWTMiddleware_NoCookie_StillCallsNext(t *testing.T) {
	h := newTestHandler(t)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.JWTMiddleware(next).ServeHTTP(rr, req)

	if !called {
		t.Error("next should be called even without a JWT cookie")
	}
}

// ─── HandleIndex ─────────────────────────────────────────────────────────────

func TestHandleIndex_Root_Returns200(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.HandleIndex(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "DevTicket") {
		t.Error("expected 'DevTicket' in response body")
	}
}

func TestHandleIndex_NonRoot_Returns404(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rr := httptest.NewRecorder()

	h.HandleIndex(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("non-root path should return 404, got %d", rr.Code)
	}
}

func TestHandleIndex_KanbanDefault(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.HandleIndex(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, "board-content") {
		t.Error("expected board-content element in kanban view")
	}
}

func TestHandleIndex_TableView(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/?view=table", nil)
	rr := httptest.NewRecorder()
	h.HandleIndex(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("table view: got %d, want %d", rr.Code, http.StatusOK)
	}
}

// ─── HandleTickets ────────────────────────────────────────────────────────────

func TestHandleTickets_KanbanView(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tickets?view=kanban", nil)
	rr := httptest.NewRecorder()

	h.HandleTickets(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "board-content") {
		t.Error("expected board-content in kanban view")
	}
}

func TestHandleTickets_TableView(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tickets?view=table", nil)
	rr := httptest.NewRecorder()

	h.HandleTickets(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "board-content") {
		t.Error("expected board-content in table view")
	}
}

func TestHandleTickets_WithFilters(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tickets?status=In+Progress&priority=High&type=Bug&component=Backend", nil)
	rr := httptest.NewRecorder()

	h.HandleTickets(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
}

// ─── HandleTicketDetail ───────────────────────────────────────────────────────

func TestHandleTicketDetail_ValidID(t *testing.T) {
	h := newTestHandler(t)
	id := firstTicketNumID(t, h)

	req := httptest.NewRequest(http.MethodGet, "/tickets/1", nil)
	req.SetPathValue("id", strings.TrimSpace(strings.Split(strings.TrimRight(strings.TrimLeft(
		strings.Replace(req.URL.Path, "/tickets/", "", 1), " "), " "), "?")[0]))

	// Use the actual numeric ID
	req2 := httptest.NewRequest(http.MethodGet, "/tickets/", nil)
	req2.SetPathValue("id", itoa(id))
	rr := httptest.NewRecorder()

	h.HandleTicketDetail(rr, req2)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "detail-modal") {
		t.Error("expected detail-modal in response")
	}
}

func TestHandleTicketDetail_InvalidID(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tickets/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()

	h.HandleTicketDetail(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleTicketDetail_NotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tickets/99999", nil)
	req.SetPathValue("id", "99999")
	rr := httptest.NewRecorder()

	h.HandleTicketDetail(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// ─── HandleNewTicketForm ──────────────────────────────────────────────────────

func TestHandleNewTicketForm_Returns200(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tickets/new", nil)
	rr := httptest.NewRecorder()

	h.HandleNewTicketForm(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "form-modal") {
		t.Error("expected form-modal in response")
	}
}

// ─── HandleEditTicketForm ─────────────────────────────────────────────────────

func TestHandleEditTicketForm_ValidID(t *testing.T) {
	h := newTestHandler(t)
	id := firstTicketNumID(t, h)

	req := httptest.NewRequest(http.MethodGet, "/tickets/1/edit", nil)
	req.SetPathValue("id", itoa(id))
	req = adminCtx(req)
	rr := httptest.NewRecorder()

	h.HandleEditTicketForm(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "form-modal") {
		t.Error("expected form-modal in response")
	}
}

func TestHandleEditTicketForm_InvalidID(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tickets/xyz/edit", nil)
	req.SetPathValue("id", "xyz")
	rr := httptest.NewRecorder()

	h.HandleEditTicketForm(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// ─── HandleCreateTicket ───────────────────────────────────────────────────────

func TestHandleCreateTicket_ValidForm(t *testing.T) {
	h := newTestHandler(t)

	form := url.Values{}
	form.Set("title", "My New Ticket")
	form.Set("description", "Some description")
	form.Set("type", "Feature")
	form.Set("priority", "High")
	form.Set("status", "Backlog")
	form.Set("assignee", "alice")
	form.Set("reporter", "bob")
	form.Set("component", "Frontend")
	form.Set("story_points", "3")

	req := httptest.NewRequest(http.MethodPost, "/tickets", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCreateTicket(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Header().Get("HX-Trigger") == "" {
		t.Error("expected HX-Trigger header after ticket creation")
	}
	if !strings.Contains(rr.Body.String(), "My New Ticket") {
		t.Error("expected ticket title in success toast")
	}
}

func TestHandleCreateTicket_Defaults(t *testing.T) {
	h := newTestHandler(t)

	// Submit without optional fields — handler should fill defaults
	form := url.Values{}
	form.Set("title", "Minimal Ticket")

	req := httptest.NewRequest(http.MethodPost, "/tickets", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCreateTicket(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleCreateTicket_AnonymousReporter(t *testing.T) {
	h := newTestHandler(t)

	form := url.Values{}
	form.Set("title", "Anon Ticket")
	req := httptest.NewRequest(http.MethodPost, "/tickets", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCreateTicket(rr, req)

	// Should succeed even without a logged-in user
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
}

// ─── HandleUpdateTicket ───────────────────────────────────────────────────────

func TestHandleUpdateTicket_ValidForm(t *testing.T) {
	h := newTestHandler(t)
	id := firstTicketNumID(t, h)

	form := url.Values{}
	form.Set("title", "Updated via Test")
	form.Set("description", "Updated desc")
	form.Set("type", "Task")
	form.Set("priority", "Medium")
	form.Set("status", "In Progress")
	form.Set("assignee", "carol")
	form.Set("reporter", "dave")
	form.Set("component", "DevOps")
	form.Set("story_points", "5")

	req := httptest.NewRequest(http.MethodPost, "/tickets/1/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", itoa(id))
	req = adminCtx(req)
	rr := httptest.NewRecorder()

	h.HandleUpdateTicket(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Header().Get("HX-Trigger"), "refresh-board") {
		t.Error("expected refresh-board in HX-Trigger")
	}
}

func TestHandleUpdateTicket_InvalidID(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/tickets/notanid/edit", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "notanid")
	rr := httptest.NewRecorder()

	h.HandleUpdateTicket(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// ─── HandleUpdateStatus ───────────────────────────────────────────────────────

func TestHandleUpdateStatus_Valid(t *testing.T) {
	h := newTestHandler(t)
	id := firstTicketNumID(t, h)

	form := url.Values{}
	form.Set("status", "Done")

	req := httptest.NewRequest(http.MethodPost, "/tickets/1/status", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", itoa(id))
	req = adminCtx(req)
	rr := httptest.NewRecorder()

	h.HandleUpdateStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Header().Get("HX-Trigger"), "refresh-board") {
		t.Error("expected refresh-board in HX-Trigger")
	}
}

func TestHandleUpdateStatus_InvalidID(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/tickets/abc/status", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()

	h.HandleUpdateStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// ─── HandleDeleteTicket ───────────────────────────────────────────────────────

func TestHandleDeleteTicket_Valid(t *testing.T) {
	h := newTestHandler(t)
	id := firstTicketNumID(t, h)

	req := httptest.NewRequest(http.MethodDelete, "/tickets/1", nil)
	req.SetPathValue("id", itoa(id))
	req = adminCtx(req)
	rr := httptest.NewRecorder()

	h.HandleDeleteTicket(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Header().Get("HX-Trigger"), "refresh-board") {
		t.Error("expected refresh-board in HX-Trigger")
	}
}

func TestHandleDeleteTicket_InvalidID(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodDelete, "/tickets/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()

	h.HandleDeleteTicket(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// ─── HandleAddComment ─────────────────────────────────────────────────────────

func TestHandleAddComment_Valid(t *testing.T) {
	h := newTestHandler(t)
	id := firstTicketNumID(t, h)

	form := url.Values{}
	form.Set("author", "tester")
	form.Set("content", "This is a great comment")

	req := httptest.NewRequest(http.MethodPost, "/tickets/1/comments", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", itoa(id))
	rr := httptest.NewRecorder()

	h.HandleAddComment(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "comment-list") {
		t.Error("expected comment-list in response")
	}
}

func TestHandleAddComment_AnonymousAuthor(t *testing.T) {
	h := newTestHandler(t)
	id := firstTicketNumID(t, h)

	form := url.Values{}
	form.Set("content", "Anonymous comment")

	req := httptest.NewRequest(http.MethodPost, "/tickets/1/comments", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", itoa(id))
	rr := httptest.NewRecorder()

	h.HandleAddComment(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHandleAddComment_InvalidID(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/tickets/bad/comments", nil)
	req.SetPathValue("id", "bad")
	rr := httptest.NewRecorder()

	h.HandleAddComment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// ─── HandleStats ──────────────────────────────────────────────────────────────

func TestHandleStats_Returns200(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tickets/stats", nil)
	rr := httptest.NewRecorder()

	h.HandleStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "stats-container") {
		t.Error("expected stats-container in response")
	}
}

// ─── HandleLoginForm ─────────────────────────────────────────────────────────

func TestHandleLoginForm_Returns200(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	h.HandleLoginForm(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "login-modal") {
		t.Error("expected login-modal in response")
	}
}

// ─── HandleLogin ─────────────────────────────────────────────────────────────

func TestHandleLogin_ValidCredentials(t *testing.T) {
	h := newTestHandler(t)

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "admin123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	// Should set a jwt_token cookie
	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "jwt_token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected jwt_token cookie to be set on successful login")
	}
	if rr.Header().Get("HX-Refresh") != "true" {
		t.Error("expected HX-Refresh: true on login")
	}
}

func TestHandleLogin_InvalidCredentials(t *testing.T) {
	h := newTestHandler(t)

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleLogin(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if strings.Contains(rr.Body.String(), "jwt_token") {
		t.Error("should not set jwt_token on failed login")
	}
}

func TestHandleLogin_UnknownUser(t *testing.T) {
	h := newTestHandler(t)

	form := url.Values{}
	form.Set("username", "ghost")
	form.Set("password", "doesnotmatter")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleLogin(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("unknown user should return 401, got %d", rr.Code)
	}
}

// ─── HandleLogout ─────────────────────────────────────────────────────────────

func TestHandleLogout_ClearsCookie(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rr := httptest.NewRecorder()

	h.HandleLogout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "jwt_token" && c.Value == "" {
			found = true
		}
	}
	if !found {
		t.Error("expected empty jwt_token cookie to be set on logout")
	}
	if rr.Header().Get("HX-Refresh") != "true" {
		t.Error("expected HX-Refresh: true on logout")
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

var itoa = func(id int64) string {
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(string([]byte{byte('0' + id)}), "", "", 1), ""), "")
}
