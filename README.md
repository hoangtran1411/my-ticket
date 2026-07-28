# 🎫 DevTicket Pro

> A high-performance, lightweight Developer Ticket & Issue Management System built with **Go**, **SQLite** (CGO-free), and standard library templates.

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)
![Database](https://img.shields.io/badge/Database-SQLite-003B57?style=flat&logo=sqlite)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

---

## 📌 Overview

**DevTicket Pro** is an agile ticket management web application designed for developer teams. It provides an intuitive interface for managing software issues, tracking bugs, planning features, and monitoring sprint progress.

Built using Go's modern `net/http` routing capabilities, DevTicket Pro runs as a single compiled binary with zero external runtime dependencies and embedded CGO-free SQLite storage.

---

## ✨ Key Features

- 📋 **Kanban Board & List Views**: Dynamic Kanban board organized by status (`To Do`, `In Progress`, `In Review`, `Done`) alongside a comprehensive tabular list view.
- 🔍 **Real-Time Filtering & Search**: Filter tickets instantly by **Status**, **Priority**, **Type** (Bug, Feature, Task, Improvement), **Component**, or text search.
- 📊 **Metrics Dashboard**: Live summary statistics displaying total tickets, story points, and status breakdowns.
- 💬 **Ticket Lifecycle & Comments**: Detailed ticket view supporting threaded discussion comments and metadata updates.
- 🔐 **JWT Authentication & Authorization**:
  - **Public Access**: Visitors can browse tickets, filter tasks, and submit new issue reports anonymously.
  - **Admin Control**: Authenticated administrators can edit ticket parameters, update lifecycle status, and delete tickets.
- ⚡ **CGO-Free SQLite Database**: Powered by `modernc.org/sqlite`, allowing effortless cross-compilation across platforms without external C compilers.
- 🌱 **Automatic Database Seeding**: Pre-populates sample developer tickets on initial startup if no database is present.

---

## 🛠️ Tech Stack

| Component | Technology | Description |
| :--- | :--- | :--- |
| **Backend** | [Go](https://go.dev/) (1.22+) | Core server using standard `net/http` router & `html/template` |
| **Database** | [modernc.org/sqlite](https://modernc.org/sqlite) | Pure Go CGO-free embedded SQLite database engine |
| **Authentication** | [golang-jwt/jwt](https://github.com/golang-jwt/jwt) | Secure HTTP-only cookie-based JWT session management |
| **Frontend** | HTML5 / CSS3 / JavaScript | Modern UI styled with Tailwind CSS (via CDN) & Custom CSS |

---

## 📂 Project Architecture

```
my-ticket/
├── main.go               # Application entry point, HTTP routes & middleware initialization
├── Makefile              # Automation task runner for build, test, and dev workflows
├── go.mod / go.sum       # Go module definitions and dependency lock files
├── db/
│   └── sqlite.go         # SQLite database initialization, table creation & seeding
├── handlers/
│   ├── handlers.go       # HTTP handler implementations & JWT middleware
│   └── handlers_test.go  # HTTP handler unit tests
├── models/
│   ├── ticket.go         # Ticket & Comment domain models and SQL query logic
│   ├── user.go           # User authentication schema & validation
│   └── ticket_test.go    # Data model unit tests
├── templates/            # Go HTML templates
│   ├── layout.html       # Master layout wrapper
│   ├── index.html        # Main dashboard page
│   ├── kanban_board.html # Kanban view component
│   ├── ticket_table.html # Tabular list view component
│   ├── ticket_card.html  # Ticket card template
│   ├── ticket_detail_modal.html # Ticket details dialog
│   ├── ticket_form_modal.html   # Ticket creation / edit form dialog
│   ├── login_modal.html         # Admin login modal
│   ├── stats_cards.html         # Summary statistics section
│   └── comment_list.html        # Comment thread component
└── static/
    └── styles.css        # Custom styles and UI overrides
```

---

## 🚀 Quick Start & Setup Tutorial

### Prerequisites

Ensure you have the following installed on your system:
- **Go**: Version 1.22 or higher ([Download Go](https://go.dev/dl/))
- **Make**: *(Optional, recommended)* Build automation tool (pre-installed on Linux/macOS or via MinGW/Chocolatey on Windows)

### 1. Clone the Repository

```bash
git clone https://github.com/hoangtran1411/my-ticket.git
cd my-ticket
```

### 2. Run the Application

You can start the development server using either Go CLI directly or Makefile commands:

#### Option A: Using Makefile (Recommended)
```bash
make run
```

#### Option B: Using Go CLI
```bash
go run main.go
```

The application will start on `http://localhost:8080` and automatically create and seed the `tickets.db` SQLite file if it does not already exist.

---

## 🔑 Authentication & Admin Credentials

DevTicket Pro includes a built-in default administrator account for performing protected actions (editing tickets, changing statuses, deleting entries).

| Property | Default Value |
| :--- | :--- |
| **Username** | `admin` |
| **Password** | `admin123` |

> **Note**: To log in as Admin, click the **Login** button in the top navigation bar and enter the credentials above.

---

## 📋 Available Makefile Commands

| Command | Description |
| :--- | :--- |
| `make run` / `make dev` | Run the application server directly |
| `make build` | Compile the binary executable (`devticket.exe` / `devticket`) |
| `make test` | Run unit tests across all packages |
| `make fmt` | Format all Go source files (`go fmt ./...`) |
| `make reset-db` | Remove `tickets.db` to re-seed sample data on next start |
| `make clean` | Remove compiled binary executables |
| `make help` | Display available commands summary |

---

## ⚙️ Environment Variables

The server configurable options via environment variables:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port number on which the HTTP server listens |

#### Example: Running on a Custom Port

- **Linux / macOS**:
  ```bash
  PORT=9000 go run main.go
  ```
- **Windows (PowerShell)**:
  ```powershell
  $env:PORT="9000"; go run main.go
  ```

---

## 🧪 Running Tests

Execute the full suite of unit tests using:

```bash
go test -v ./...
```

Or via Makefile:

```bash
make test
```

---

## 📄 API & HTTP Routes Overview

| Method | Path | Access | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/` | Public | Main dashboard |
| `GET` | `/tickets` | Public | List & search tickets (HTML snippet / full page) |
| `GET` | `/tickets/stats` | Public | Retrieve summary statistics cards |
| `GET` | `/tickets/new` | Public | Get creation modal form |
| `POST` | `/tickets` | Public | Create a new ticket |
| `GET` | `/tickets/{id}` | Public | View ticket detail modal & comments |
| `POST` | `/tickets/{id}/comments` | Public | Add a comment to a ticket |
| `GET` | `/login` | Public | Get login modal form |
| `POST` | `/login` | Public | Authenticate user & receive JWT cookie |
| `POST` | `/logout` | Public | Clear session cookie |
| `GET` | `/tickets/{id}/edit` | 🔒 Admin | Get edit ticket modal form |
| `POST` | `/tickets/{id}/edit` | 🔒 Admin | Update existing ticket details |
| `POST` | `/tickets/{id}/status` | 🔒 Admin | Move ticket status on Kanban board |
| `DELETE` | `/tickets/{id}` | 🔒 Admin | Delete a ticket |

---

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
