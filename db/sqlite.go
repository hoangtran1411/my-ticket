package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"my-ticket/models"

	_ "modernc.org/sqlite"
)

func InitDB(filepath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	createTablesQuery := `
	CREATE TABLE IF NOT EXISTS tickets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ticket_key TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		type TEXT NOT NULL,
		priority TEXT NOT NULL,
		status TEXT NOT NULL,
		assignee TEXT NOT NULL,
		reporter TEXT NOT NULL,
		component TEXT NOT NULL,
		story_points INTEGER DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ticket_id INTEGER NOT NULL,
		author TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE
	);
	`

	_, err = database.Exec(createTablesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	if err := models.CreateUserTable(database); err != nil {
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	// Seed data if empty
	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM tickets").Scan(&count)
	if err == nil && count == 0 {
		seedDatabase(database)
	}

	return database, nil
}

func seedDatabase(db *sql.DB) {
	log.Println("Seeding initial development tickets into SQLite database...")
	now := time.Now()

	sampleTickets := []struct {
		key         string
		title       string
		description string
		typ         string
		priority    string
		status      string
		assignee    string
		reporter    string
		component   string
		points      int
		daysAgo     int
	}{
		{
			key:         "DEV-001",
			title:       "Fix JWT Refresh Token Expiration & Cookie Handling",
			description: "Users are getting prematurely logged out when refreshing tabs. Need to implement sliding session window and secure HttpOnly cookie storage for refresh tokens.",
			typ:         "Bug",
			priority:    "Urgent",
			status:      "In Progress",
			assignee:    "Alex Rivers",
			reporter:    "Sarah Connor",
			component:   "Backend",
			points:      5,
			daysAgo:     2,
		},
		{
			key:         "DEV-002",
			title:       "Implement Dynamic Kanban Board Drag & Drop",
			description: "Enable drag and drop card reordering with HTMX swap triggers. Columns must update task status immediately in SQLite with smooth animation.",
			typ:         "Feature",
			priority:    "High",
			status:      "In Progress",
			assignee:    "Devon Vance",
			reporter:    "Alex Rivers",
			component:   "Frontend",
			points:      8,
			daysAgo:     3,
		},
		{
			key:         "DEV-003",
			title:       "Optimize SQLite WAL Mode & Connection Pool",
			description: "Enable Write-Ahead Logging (WAL) pragma mode to allow concurrent readers and prevent database locking errors during high concurrency.",
			typ:         "Refactor",
			priority:    "High",
			status:      "Backlog",
			assignee:    "Marcus Vance",
			reporter:    "Lead Architect",
			component:   "Database",
			points:      3,
			daysAgo:     5,
		},
		{
			key:         "DEV-004",
			title:       "Audit API Security Headers & CORS Policy",
			description: "Add Content Security Policy (CSP), Strict-Transport-Security (HSTS), and validate XSS prevention across dynamic HTML template rendering.",
			typ:         "Security",
			priority:    "Urgent",
			status:      "In Review",
			assignee:    "Elena Rostova",
			reporter:    "SecOps Team",
			component:   "DevOps",
			points:      5,
			daysAgo:     1,
		},
		{
			key:         "DEV-005",
			title:       "Create Dark Mode UI Palette & Tailwind Components",
			description: "Design sleek, high-contrast dark theme with glassmorphism modal panels, badge gradients, and smooth hover states.",
			typ:         "Feature",
			priority:    "Medium",
			status:      "Done",
			assignee:    "Devon Vance",
			reporter:    "Product Owner",
			component:   "UI/UX",
			points:      3,
			daysAgo:     6,
		},
		{
			key:         "DEV-006",
			title:       "Setup Automated CI/CD Pipeline for Go Binary Build",
			description: "Configure GitHub Actions workflow to build multi-arch binaries (Linux, Windows, macOS) and run unit test suites on PR push.",
			typ:         "Task",
			priority:    "Medium",
			status:      "Backlog",
			assignee:    "Marcus Vance",
			reporter:    "DevOps Lead",
			component:   "DevOps",
			points:      5,
			daysAgo:     4,
		},
		{
			key:         "DEV-007",
			title:       "Real-time Search & Multi-Column Filtering",
			description: "Add instant keypress search with 300ms debounce using HTMX `hx-get` targeting the task table and board fragments.",
			typ:         "Feature",
			priority:    "High",
			status:      "In Review",
			assignee:    "Sarah Connor",
			reporter:    "Devon Vance",
			component:   "Frontend",
			points:      5,
			daysAgo:     2,
		},
		{
			key:         "DEV-008",
			title:       "Fix Memory Leak in WebSockets Event Dispatcher",
			description: "Goroutines servicing disconnected client sockets are not closing channel handles cleanly causing gradual RSS memory growth.",
			typ:         "Bug",
			priority:    "Urgent",
			status:      "Done",
			assignee:    "Alex Rivers",
			reporter:    "QA Engineer",
			component:   "Backend",
			points:      8,
			daysAgo:     7,
		},
	}

	for _, t := range sampleTickets {
		createdAt := now.AddDate(0, 0, -t.daysAgo).Format("2006-01-02 15:04:05")
		res, err := db.Exec(
			`INSERT INTO tickets (ticket_key, title, description, type, priority, status, assignee, reporter, component, story_points, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.key, t.title, t.description, t.typ, t.priority, t.status, t.assignee, t.reporter, t.component, t.points, createdAt, createdAt,
		)
		if err != nil {
			log.Printf("Failed to seed ticket %s: %v", t.key, err)
			continue
		}

		ticketID, err := res.LastInsertId()
		if err == nil {
			// Seed sample comments
			_, _ = db.Exec(
				`INSERT INTO comments (ticket_id, author, content, created_at) VALUES (?, ?, ?, ?)`,
				ticketID, t.reporter, fmt.Sprintf("Initial bug report logged for %s. Assigned to %s.", t.key, t.assignee), createdAt,
			)
			if t.status == "In Progress" || t.status == "In Review" {
				_, _ = db.Exec(
					`INSERT INTO comments (ticket_id, author, content, created_at) VALUES (?, ?, ?, ?)`,
					ticketID, t.assignee, "Working on reproducing and fixing this issue. Patch submitted for review.", now.Format("2006-01-02 15:04:05"),
				)
			}
		}
	}
	log.Println("Seeding completed successfully!")
}
