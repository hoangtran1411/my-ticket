package models

import (
	"testing"
	"time"
)

func TestStatusClass(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"Backlog", "bg-slate-800/80 text-slate-300 border-slate-700"},
		{"In Progress", "bg-amber-500/10 text-amber-400 border-amber-500/30"},
		{"In Review", "bg-indigo-500/10 text-indigo-400 border-indigo-500/30"},
		{"Done", "bg-emerald-500/10 text-emerald-400 border-emerald-500/30"},
	}

	for _, tt := range tests {
		result := StatusClass(tt.status)
		if result != tt.expected {
			t.Errorf("StatusClass(%q) = %q; want %q", tt.status, result, tt.expected)
		}
	}
}

func TestShortDescription(t *testing.T) {
	shortText := "Short text"
	ticketShort := Ticket{Description: shortText}
	if ticketShort.ShortDescription() != shortText {
		t.Errorf("Expected unchanged text for short description, got %q", ticketShort.ShortDescription())
	}

	longText := "This is a very long description that exceeds one hundred and ten characters limit and should be truncated cleanly with an ellipsis..."
	ticketLong := Ticket{Description: longText}
	result := ticketLong.ShortDescription()
	if len(result) > 110 {
		t.Errorf("Expected truncated string length <= 110, got %d", len(result))
	}
}

func TestFormattedDate(t *testing.T) {
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)
	ticket := Ticket{CreatedAt: now}
	if ticket.FormattedDate() != "Jul 28, 2026" {
		t.Errorf("Expected 'Jul 28, 2026', got %q", ticket.FormattedDate())
	}
}
