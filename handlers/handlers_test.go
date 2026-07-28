package handlers

import (
	"html/template"
	"testing"

	"my-ticket/models"
)

func TestParseTemplates(t *testing.T) {
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

	_, err := template.New("").Funcs(funcMap).ParseGlob("../templates/*.html")
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}
}
