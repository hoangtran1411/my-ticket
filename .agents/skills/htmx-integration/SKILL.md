---
name: htmx-integration
description: Best practices for implementing HTMX dynamic UI components in Go net/http web apps.
---

# HTMX Integration Skill for Go Web Applications

This skill provides patterns and conventions for adding fast, zero-reload interactive features (Kanban drag-and-drop, debounced search, modal dialogs, and dynamic comments) using HTMX and Go standard `html/template`.

## Core Conventions

1. **Trigger Headers**:
   - Use `HX-Trigger` header in Go handlers to notify client-side components to refresh (e.g. `w.Header().Set("HX-Trigger", "refresh-board, refresh-stats, close-modal")`).

2. **Partial HTML Fragment Rendering**:
   - Return rendered HTML fragments (e.g. `kanban_board.html`, `ticket_table.html`, `comment_list.html`) for `hx-get` or `hx-post` requests instead of full layout pages.

3. **Debounced Search**:
   - Apply `hx-get="/tickets"` with `hx-trigger="keyup changed delay:300ms, search"` on input filters.

4. **Response Toasts**:
   - Return clean toast notification HTML snippets for instant feedback when creating/updating tickets.
