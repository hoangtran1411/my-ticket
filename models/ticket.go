package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	StatusBacklog    = "Backlog"
	StatusInProgress = "In Progress"
	StatusInReview   = "In Review"
	StatusDone       = "Done"
)

type Ticket struct {
	ID          int64     `json:"id"`
	TicketKey   string    `json:"ticket_key"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"`        // Bug, Feature, Refactor, Task, Security
	Priority    string    `json:"priority"`    // Low, Medium, High, Urgent
	Status      string    `json:"status"`      // Backlog, In Progress, In Review, Done
	Assignee    string    `json:"assignee"`
	Reporter    string    `json:"reporter"`
	Component   string    `json:"component"`   // Frontend, Backend, Database, DevOps, API
	StoryPoints int       `json:"story_points"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Comments    []Comment `json:"comments,omitempty"`
}

type Comment struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type TicketFilter struct {
	Query     string
	Status    string
	Priority  string
	Type      string
	Component string
	ViewMode  string // "kanban" or "table"
}

type TicketStats struct {
	Total          int
	Backlog        int
	InProgress     int
	InReview       int
	Done           int
	UrgentCount    int
	CompletionRate int
}

func (t Ticket) FormattedDate() string {
	return t.CreatedAt.Format("Jan 02, 2006")
}

func (t Ticket) ShortDescription() string {
	if len(t.Description) > 110 {
		return t.Description[:107] + "..."
	}
	return t.Description
}

func (c Comment) FormattedTime() string {
	return c.CreatedAt.Format("Jan 02, 15:04")
}

func ParseTime(str string) time.Time {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, str); err == nil {
			return t
		}
	}
	return time.Time{}
}

const filterAll = "All"

func GetTickets(db *sql.DB, filter TicketFilter) ([]Ticket, error) {
	query := `SELECT id, ticket_key, title, description, type, priority, status, assignee, reporter, component, story_points, created_at, updated_at FROM tickets WHERE 1=1`
	var args []interface{}

	if filter.Query != "" {
		query += ` AND (title LIKE ? OR description LIKE ? OR ticket_key LIKE ? OR assignee LIKE ?)`
		likeQuery := "%" + filter.Query + "%"
		args = append(args, likeQuery, likeQuery, likeQuery, likeQuery)
	}

	if filter.Status != "" && filter.Status != filterAll {
		query += ` AND status = ?`
		args = append(args, filter.Status)
	}

	if filter.Priority != "" && filter.Priority != filterAll {
		query += ` AND priority = ?`
		args = append(args, filter.Priority)
	}

	if filter.Type != "" && filter.Type != filterAll {
		query += ` AND type = ?`
		args = append(args, filter.Type)
	}

	if filter.Component != "" && filter.Component != filterAll {
		query += ` AND component = ?`
		args = append(args, filter.Component)
	}

	query += ` ORDER BY updated_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tickets []Ticket
	for rows.Next() {
		var t Ticket
		var createdAtStr, updatedAtStr string
		err := rows.Scan(
			&t.ID, &t.TicketKey, &t.Title, &t.Description, &t.Type, &t.Priority,
			&t.Status, &t.Assignee, &t.Reporter, &t.Component, &t.StoryPoints,
			&createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}
		t.CreatedAt = ParseTime(createdAtStr)
		t.UpdatedAt = ParseTime(updatedAtStr)
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func GetTicketByID(db *sql.DB, id int64) (*Ticket, error) {
	query := `SELECT id, ticket_key, title, description, type, priority, status, assignee, reporter, component, story_points, created_at, updated_at FROM tickets WHERE id = ?`
	row := db.QueryRow(query, id)

	var t Ticket
	var createdAtStr, updatedAtStr string
	err := row.Scan(
		&t.ID, &t.TicketKey, &t.Title, &t.Description, &t.Type, &t.Priority,
		&t.Status, &t.Assignee, &t.Reporter, &t.Component, &t.StoryPoints,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	t.CreatedAt = ParseTime(createdAtStr)
	t.UpdatedAt = ParseTime(updatedAtStr)

	comments, err := GetCommentsByTicketID(db, id)
	if err == nil {
		t.Comments = comments
	}

	return &t, nil
}

func CreateTicket(db *sql.DB, t *Ticket) error {
	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")

	// Generate Ticket Key DEV-xxx
	var lastID int64
	err := db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM tickets").Scan(&lastID)
	if err != nil {
		lastID = 0
	}
	t.TicketKey = fmt.Sprintf("DEV-%03d", lastID+1)

	query := `INSERT INTO tickets (ticket_key, title, description, type, priority, status, assignee, reporter, component, story_points, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := db.Exec(query, t.TicketKey, t.Title, t.Description, t.Type, t.Priority, t.Status, t.Assignee, t.Reporter, t.Component, t.StoryPoints, nowStr, nowStr)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = id
	// Populate timestamps on the struct so callers display the correct time
	t.CreatedAt = now
	t.UpdatedAt = now
	return nil
}

func UpdateTicket(db *sql.DB, t *Ticket) error {
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	query := `UPDATE tickets SET title = ?, description = ?, type = ?, priority = ?, status = ?, assignee = ?, reporter = ?, component = ?, story_points = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, t.Title, t.Description, t.Type, t.Priority, t.Status, t.Assignee, t.Reporter, t.Component, t.StoryPoints, nowStr, t.ID)
	return err
}

func UpdateTicketStatus(db *sql.DB, id int64, newStatus string) error {
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	query := `UPDATE tickets SET status = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, newStatus, nowStr, id)
	return err
}

func DeleteTicket(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM comments WHERE ticket_id = ?", id)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM tickets WHERE id = ?", id)
	return err
}

func AddComment(db *sql.DB, comment *Comment) error {
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	query := `INSERT INTO comments (ticket_id, author, content, created_at) VALUES (?, ?, ?, ?)`
	res, err := db.Exec(query, comment.TicketID, comment.Author, comment.Content, nowStr)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	comment.ID = id
	comment.CreatedAt = time.Now()
	return nil
}

func GetCommentsByTicketID(db *sql.DB, ticketID int64) ([]Comment, error) {
	query := `SELECT id, ticket_id, author, content, created_at FROM comments WHERE ticket_id = ? ORDER BY created_at ASC`
	rows, err := db.Query(query, ticketID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var comments []Comment
	for rows.Next() {
		var c Comment
		var createdAtStr string
		if err := rows.Scan(&c.ID, &c.TicketID, &c.Author, &c.Content, &createdAtStr); err != nil {
			return nil, err
		}
		c.CreatedAt = ParseTime(createdAtStr)
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

func GetStats(db *sql.DB) (TicketStats, error) {
	var stats TicketStats
	err := db.QueryRow(`SELECT COUNT(*) FROM tickets`).Scan(&stats.Total)
	if err != nil {
		return stats, err
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM tickets WHERE status = 'Backlog'`).Scan(&stats.Backlog)
	_ = db.QueryRow(`SELECT COUNT(*) FROM tickets WHERE status = 'In Progress'`).Scan(&stats.InProgress)
	_ = db.QueryRow(`SELECT COUNT(*) FROM tickets WHERE status = 'In Review'`).Scan(&stats.InReview)
	_ = db.QueryRow(`SELECT COUNT(*) FROM tickets WHERE status = 'Done'`).Scan(&stats.Done)
	_ = db.QueryRow(`SELECT COUNT(*) FROM tickets WHERE priority = 'Urgent' AND status != 'Done'`).Scan(&stats.UrgentCount)

	if stats.Total > 0 {
		stats.CompletionRate = (stats.Done * 100) / stats.Total
	}
	return stats, nil
}

func StatusClass(status string) string {
	switch strings.ToLower(status) {
	case "backlog":
		return "bg-slate-800/80 text-slate-300 border-slate-700"
	case "in progress":
		return "bg-amber-500/10 text-amber-400 border-amber-500/30"
	case "in review":
		return "bg-indigo-500/10 text-indigo-400 border-indigo-500/30"
	case "done":
		return "bg-emerald-500/10 text-emerald-400 border-emerald-500/30"
	default:
		return "bg-slate-800 text-slate-300 border-slate-700"
	}
}

func PriorityClass(priority string) string {
	switch strings.ToLower(priority) {
	case "urgent":
		return "bg-rose-500/15 text-rose-400 border-rose-500/30"
	case "high":
		return "bg-orange-500/15 text-orange-400 border-orange-500/30"
	case "medium":
		return "bg-sky-500/15 text-sky-400 border-sky-500/30"
	case "low":
		return "bg-slate-500/15 text-slate-400 border-slate-500/30"
	default:
		return "bg-slate-500/15 text-slate-400 border-slate-500/30"
	}
}

func TypeIcon(t string) string {
	switch strings.ToLower(t) {
	case "bug":
		return "fa-bug text-rose-400"
	case "feature":
		return "fa-rocket text-indigo-400"
	case "refactor":
		return "fa-code-branch text-amber-400"
	case "security":
		return "fa-shield-halved text-purple-400"
	default:
		return "fa-list-check text-cyan-400"
	}
}
