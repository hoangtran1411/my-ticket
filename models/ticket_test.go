package models

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// ─── Test DB helpers ─────────────────────────────────────────────────────────

// setupTestDB opens an in-memory SQLite DB with the full schema.
// Shared by all *_test.go files in this package.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory DB: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tickets (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			ticket_key    TEXT     NOT NULL UNIQUE,
			title         TEXT     NOT NULL,
			description   TEXT     NOT NULL,
			type          TEXT     NOT NULL,
			priority      TEXT     NOT NULL,
			status        TEXT     NOT NULL,
			assignee      TEXT     NOT NULL,
			reporter      TEXT     NOT NULL,
			component     TEXT     NOT NULL,
			story_points  INTEGER  DEFAULT 1,
			created_at    DATETIME NOT NULL,
			updated_at    DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS comments (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			ticket_id  INTEGER  NOT NULL,
			author     TEXT     NOT NULL,
			content    TEXT     NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS users (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			username      TEXT     NOT NULL UNIQUE,
			password_hash TEXT     NOT NULL,
			role          TEXT     NOT NULL,
			created_at    DATETIME NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// seedTicket inserts a ticket and returns it with its generated ID.
func seedTicket(t *testing.T, db *sql.DB) *Ticket {
	t.Helper()
	tk := &Ticket{
		Title:       "Test Ticket",
		Description: "A test description",
		Type:        "Bug",
		Priority:    "High",
		Status:      "Backlog",
		Assignee:    "alice",
		Reporter:    "bob",
		Component:   "Backend",
		StoryPoints: 5,
	}
	if err := CreateTicket(db, tk); err != nil {
		t.Fatalf("seedTicket: %v", err)
	}
	return tk
}

// ─── Pure-function tests ──────────────────────────────────────────────────────

func TestStatusClass(t *testing.T) {
	cases := []struct{ status, want string }{
		{"Backlog", "bg-slate-800/80 text-slate-300 border-slate-700"},
		{"In Progress", "bg-amber-500/10 text-amber-400 border-amber-500/30"},
		{"In Review", "bg-indigo-500/10 text-indigo-400 border-indigo-500/30"},
		{"Done", "bg-emerald-500/10 text-emerald-400 border-emerald-500/30"},
		{"DONE", "bg-emerald-500/10 text-emerald-400 border-emerald-500/30"}, // case-insensitive
		{"unknown", "bg-slate-800 text-slate-300 border-slate-700"},
	}
	for _, c := range cases {
		if got := StatusClass(c.status); got != c.want {
			t.Errorf("StatusClass(%q) = %q; want %q", c.status, got, c.want)
		}
	}
}

func TestPriorityClass(t *testing.T) {
	cases := []struct{ priority, want string }{
		{"Urgent", "bg-rose-500/15 text-rose-400 border-rose-500/30"},
		{"High", "bg-orange-500/15 text-orange-400 border-orange-500/30"},
		{"Medium", "bg-sky-500/15 text-sky-400 border-sky-500/30"},
		{"Low", "bg-slate-500/15 text-slate-400 border-slate-500/30"},
		{"URGENT", "bg-rose-500/15 text-rose-400 border-rose-500/30"}, // case-insensitive
		{"unknown", "bg-slate-500/15 text-slate-400 border-slate-500/30"},
	}
	for _, c := range cases {
		if got := PriorityClass(c.priority); got != c.want {
			t.Errorf("PriorityClass(%q) = %q; want %q", c.priority, got, c.want)
		}
	}
}

func TestTypeIcon(t *testing.T) {
	cases := []struct{ typ, want string }{
		{"Bug", "fa-bug text-rose-400"},
		{"Feature", "fa-rocket text-indigo-400"},
		{"Refactor", "fa-code-branch text-amber-400"},
		{"Security", "fa-shield-halved text-purple-400"},
		{"Task", "fa-check-square text-cyan-400"},
		{"unknown", "fa-check-square text-cyan-400"}, // default
	}
	for _, c := range cases {
		if got := TypeIcon(c.typ); got != c.want {
			t.Errorf("TypeIcon(%q) = %q; want %q", c.typ, got, c.want)
		}
	}
}

func TestShortDescription(t *testing.T) {
	t.Run("short text unchanged", func(t *testing.T) {
		tk := Ticket{Description: "Short"}
		if got := tk.ShortDescription(); got != "Short" {
			t.Errorf("got %q, want %q", got, "Short")
		}
	})
	t.Run("long text is truncated with ellipsis", func(t *testing.T) {
		tk := Ticket{Description: strings.Repeat("a", 200)}
		got := tk.ShortDescription()
		if len(got) > 110 {
			t.Errorf("len = %d, want ≤ 110", len(got))
		}
		if !strings.HasSuffix(got, "...") {
			t.Errorf("expected ellipsis suffix, got %q", got)
		}
	})
	t.Run("exactly 110 chars unchanged", func(t *testing.T) {
		text := strings.Repeat("b", 110)
		tk := Ticket{Description: text}
		if got := tk.ShortDescription(); got != text {
			t.Errorf("expected unchanged, got different result")
		}
	})
}

func TestFormattedDate(t *testing.T) {
	tk := Ticket{CreatedAt: time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)}
	if got := tk.FormattedDate(); got != "Jul 28, 2026" {
		t.Errorf("FormattedDate() = %q; want %q", got, "Jul 28, 2026")
	}
}

func TestCommentFormattedTime(t *testing.T) {
	c := Comment{CreatedAt: time.Date(2026, time.July, 28, 15, 30, 0, 0, time.UTC)}
	want := "Jul 28, 15:30"
	if got := c.FormattedTime(); got != want {
		t.Errorf("FormattedTime() = %q; want %q", got, want)
	}
}

// ─── CRUD DB tests ────────────────────────────────────────────────────────────

func TestCreateTicket_SetsIDAndTimestamps(t *testing.T) {
	db := setupTestDB(t)
	tk := seedTicket(t, db)

	if tk.ID == 0 {
		t.Error("expected non-zero ID after CreateTicket")
	}
	if tk.TicketKey == "" {
		t.Error("expected non-empty TicketKey")
	}
	if tk.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero after CreateTicket")
	}
	if tk.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero after CreateTicket")
	}
}

func TestCreateTicket_KeyFormat(t *testing.T) {
	db := setupTestDB(t)
	tk := seedTicket(t, db)
	if !strings.HasPrefix(tk.TicketKey, "DEV-") {
		t.Errorf("TicketKey %q does not start with DEV-", tk.TicketKey)
	}
}

func TestGetTickets_Empty(t *testing.T) {
	db := setupTestDB(t)
	tickets, err := GetTickets(db, TicketFilter{})
	if err != nil {
		t.Fatalf("GetTickets: %v", err)
	}
	if len(tickets) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(tickets))
	}
}

func TestGetTickets_AllFilters(t *testing.T) {
	db := setupTestDB(t)
	seedTicket(t, db) // Bug, High, Backlog, Backend

	tk2 := &Ticket{
		Title: "Feature X", Description: "desc", Type: "Feature",
		Priority: "Low", Status: "Done", Assignee: "carol",
		Reporter: "alice", Component: "Frontend", StoryPoints: 2,
	}
	_ = CreateTicket(db, tk2)

	tests := []struct {
		name   string
		filter TicketFilter
		want   int
	}{
		{"no filter returns all", TicketFilter{}, 2},
		{"status=Done", TicketFilter{Status: "Done"}, 1},
		{"status=All skipped", TicketFilter{Status: "All"}, 2},
		{"priority=High", TicketFilter{Priority: "High"}, 1},
		{"type=Bug", TicketFilter{Type: "Bug"}, 1},
		{"type=All skipped", TicketFilter{Type: "All"}, 2},
		{"component=Frontend", TicketFilter{Component: "Frontend"}, 1},
		{"component=All skipped", TicketFilter{Component: "All"}, 2},
		{"query matches title", TicketFilter{Query: "Feature X"}, 1},
		{"query matches assignee", TicketFilter{Query: "carol"}, 1},
		{"query no match", TicketFilter{Query: "zzznomatch"}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetTickets(db, tc.filter)
			if err != nil {
				t.Fatalf("GetTickets: %v", err)
			}
			if len(got) != tc.want {
				t.Errorf("got %d tickets, want %d", len(got), tc.want)
			}
		})
	}
}

func TestGetTicketByID_Found(t *testing.T) {
	db := setupTestDB(t)
	created := seedTicket(t, db)

	got, err := GetTicketByID(db, created.ID)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if got.Title != created.Title {
		t.Errorf("title: got %q, want %q", got.Title, created.Title)
	}
	if got.Component != created.Component {
		t.Errorf("component: got %q, want %q", got.Component, created.Component)
	}
}

func TestGetTicketByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	_, err := GetTicketByID(db, 9999)
	if err == nil {
		t.Error("expected error for missing ticket, got nil")
	}
}

func TestUpdateTicket(t *testing.T) {
	db := setupTestDB(t)
	tk := seedTicket(t, db)

	tk.Title = "Updated Title"
	tk.Priority = "Urgent"
	tk.Status = "In Progress"
	tk.StoryPoints = 8

	if err := UpdateTicket(db, tk); err != nil {
		t.Fatalf("UpdateTicket: %v", err)
	}

	got, err := GetTicketByID(db, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("title not updated: got %q", got.Title)
	}
	if got.Status != "In Progress" {
		t.Errorf("status not updated: got %q", got.Status)
	}
	if got.StoryPoints != 8 {
		t.Errorf("story_points not updated: got %d", got.StoryPoints)
	}
}

func TestUpdateTicketStatus(t *testing.T) {
	db := setupTestDB(t)
	tk := seedTicket(t, db)

	if err := UpdateTicketStatus(db, tk.ID, "Done"); err != nil {
		t.Fatalf("UpdateTicketStatus: %v", err)
	}
	got, err := GetTicketByID(db, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "Done" {
		t.Errorf("status not updated: got %q", got.Status)
	}
}

func TestDeleteTicket_RemovesTicketAndComments(t *testing.T) {
	db := setupTestDB(t)
	tk := seedTicket(t, db)

	// add a comment — should be cascade-deleted
	_ = AddComment(db, &Comment{TicketID: tk.ID, Author: "user", Content: "hi"})

	if err := DeleteTicket(db, tk.ID); err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}

	_, err := GetTicketByID(db, tk.ID)
	if err == nil {
		t.Error("ticket should be gone after DeleteTicket")
	}

	comments, _ := GetCommentsByTicketID(db, tk.ID)
	if len(comments) != 0 {
		t.Errorf("expected 0 comments after delete, got %d", len(comments))
	}
}

func TestGetStats(t *testing.T) {
	db := setupTestDB(t)

	for _, status := range []string{"Backlog", "In Progress", "In Review", "Done", "Done"} {
		_ = CreateTicket(db, &Ticket{
			Title: "t", Description: "d", Type: "Task", Priority: "Low",
			Status: status, Assignee: "a", Reporter: "r", Component: "Backend",
		})
	}
	// urgent, not Done
	_ = CreateTicket(db, &Ticket{
		Title: "u", Description: "d", Type: "Bug", Priority: "Urgent",
		Status: "In Progress", Assignee: "a", Reporter: "r", Component: "Backend",
	})

	stats, err := GetStats(db)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	if stats.Total != 6 {
		t.Errorf("Total: got %d, want 6", stats.Total)
	}
	if stats.Backlog != 1 {
		t.Errorf("Backlog: got %d, want 1", stats.Backlog)
	}
	if stats.InProgress != 2 {
		t.Errorf("InProgress: got %d, want 2", stats.InProgress)
	}
	if stats.InReview != 1 {
		t.Errorf("InReview: got %d, want 1", stats.InReview)
	}
	if stats.Done != 2 {
		t.Errorf("Done: got %d, want 2", stats.Done)
	}
	if stats.UrgentCount != 1 {
		t.Errorf("UrgentCount: got %d, want 1", stats.UrgentCount)
	}
	// (2/6)*100 = 33
	if stats.CompletionRate != 33 {
		t.Errorf("CompletionRate: got %d, want 33", stats.CompletionRate)
	}
}

func TestGetStats_Empty(t *testing.T) {
	db := setupTestDB(t)
	stats, err := GetStats(db)
	if err != nil {
		t.Fatalf("GetStats on empty DB: %v", err)
	}
	if stats.Total != 0 {
		t.Errorf("expected 0 total, got %d", stats.Total)
	}
	if stats.CompletionRate != 0 {
		t.Errorf("expected 0 completion rate, got %d", stats.CompletionRate)
	}
}

func TestAddComment_SetsIDAndTimestamp(t *testing.T) {
	db := setupTestDB(t)
	tk := seedTicket(t, db)

	c := &Comment{TicketID: tk.ID, Author: "tester", Content: "Hello!"}
	if err := AddComment(db, c); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if c.ID == 0 {
		t.Error("expected non-zero comment ID")
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set on comment")
	}
}

func TestGetCommentsByTicketID(t *testing.T) {
	db := setupTestDB(t)
	tk := seedTicket(t, db)

	_ = AddComment(db, &Comment{TicketID: tk.ID, Author: "alice", Content: "first"})
	_ = AddComment(db, &Comment{TicketID: tk.ID, Author: "bob", Content: "second"})

	comments, err := GetCommentsByTicketID(db, tk.ID)
	if err != nil {
		t.Fatalf("GetCommentsByTicketID: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
}

func TestGetCommentsByTicketID_Empty(t *testing.T) {
	db := setupTestDB(t)
	tk := seedTicket(t, db)

	comments, err := GetCommentsByTicketID(db, tk.ID)
	if err != nil {
		t.Fatalf("GetCommentsByTicketID: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}
