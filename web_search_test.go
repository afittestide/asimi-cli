package main

import (
	"context"
	"fmt"
	"testing"
)

func TestWebSearchTool_Name(t *testing.T) {
	tool := WebSearchTool{}
	if tool.Name() != "web_search" {
		t.Errorf("Expected name to be 'web_search', got '%s'", tool.Name())
	}
}

func TestWebSearchTool_Description(t *testing.T) {
	tool := WebSearchTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
	if !contains(desc, "DuckDuckGo") {
		t.Error("Description should mention DuckDuckGo")
	}
}

func TestWebSearchTool_CallWithJSON(t *testing.T) {
	tool := WebSearchTool{}

	// Skip test if we can't reach DuckDuckGo
	t.Skip("Skipping web search test to avoid network dependency")

	input := `{"query": "golang testing", "max_results": 3}`
	result, err := tool.Call(context.Background(), input)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}
}

func TestWebSearchTool_CallWithEmptyQuery(t *testing.T) {
	tool := WebSearchTool{}

	input := `{"query": ""}`
	_, err := tool.Call(context.Background(), input)

	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestWebSearchTool_Format(t *testing.T) {
	tool := WebSearchTool{}

	input := `{"query": "test query"}`
	result := `{"query": "test query", "results": [{"title": "Test", "url": "http://example.com", "snippet": "test"}]}`

	formatted := tool.Format(input, result, nil)

	if !contains(formatted, "Web Search") {
		t.Error("Formatted output should contain 'Web Search'")
	}
	if !contains(formatted, "test query") {
		t.Error("Formatted output should contain query")
	}
	if !contains(formatted, "1 result") {
		t.Error("Formatted output should contain result count")
	}
}

func TestWebSearchTool_FormatWithError(t *testing.T) {
	tool := WebSearchTool{}

	input := `{"query": "test"}`
	err := fmt.Errorf("search failed")

	formatted := tool.Format(input, "", err)

	if !contains(formatted, "Error") {
		t.Error("Formatted error should contain 'Error'")
	}
}

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove tags",
			input:    "<b>Bold</b> text",
			expected: "Bold text",
		},
		{
			name:     "Decode entities",
			input:    "Test &amp; example",
			expected: "Test & example",
		},
		{
			name:     "Complex HTML",
			input:    "<div>Hello &nbsp;<span>world</span>&quot;</div>",
			expected: "Hello  world\"", // Note: double space from &nbsp; and removed tags
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanHTML(tt.input)
			if result != tt.expected {
				t.Errorf("cleanHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
