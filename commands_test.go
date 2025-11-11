package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindCommand(t *testing.T) {
	registry := NewCommandRegistry()

	tests := []struct {
		name            string
		input           string
		expectFound     bool
		expectCommand   string
		expectMatches   int
		expectAmbiguous bool
	}{
		{
			name:          "exact match with colon",
			input:         ":quit",
			expectFound:   true,
			expectCommand: "/quit",
			expectMatches: 1,
		},
		{
			name:          "partial match single - q",
			input:         ":q",
			expectFound:   true,
			expectCommand: "/quit",
			expectMatches: 1,
		},
		{
			name:          "partial match single - qu",
			input:         ":qu",
			expectFound:   true,
			expectCommand: "/quit",
			expectMatches: 1,
		},
		{
			name:          "partial match single - qui",
			input:         ":qui",
			expectFound:   true,
			expectCommand: "/quit",
			expectMatches: 1,
		},
		{
			name:          "partial match single - h",
			input:         ":h",
			expectFound:   true,
			expectCommand: "/help",
			expectMatches: 1,
		},
		{
			name:          "partial match single - n",
			input:         ":n",
			expectFound:   true,
			expectCommand: "/new",
			expectMatches: 1,
		},
		{
			name:            "ambiguous match - c",
			input:           ":c",
			expectFound:     false,
			expectMatches:   2, // /clear-history, /compact, and /context
			expectAmbiguous: true,
		},
		{
			name:            "ambiguous match - co",
			input:           ":co",
			expectFound:     false,
			expectMatches:   2, // /compact and /context
			expectAmbiguous: true,
		},
		{
			name:          "partial disambiguated - com",
			input:         ":com",
			expectFound:   true,
			expectCommand: "/compact",
			expectMatches: 1,
		},
		{
			name:          "partial disambiguated - con",
			input:         ":con",
			expectFound:   true,
			expectCommand: "/context",
			expectMatches: 1,
		},
		{
			name:          "no match",
			input:         ":xyz",
			expectFound:   false,
			expectMatches: 0,
		},
		{
			name:          "empty input",
			input:         "",
			expectFound:   false,
			expectMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, matches, found := registry.FindCommand(tt.input)

			require.Equal(t, tt.expectFound, found, "found mismatch")
			require.Equal(t, tt.expectMatches, len(matches), "matches count mismatch")

			if tt.expectFound {
				require.Equal(t, tt.expectCommand, cmd.Name, "command name mismatch")
			}

			if tt.expectAmbiguous {
				require.False(t, found, "should not find unique match for ambiguous input")
				require.Greater(t, len(matches), 1, "ambiguous should have multiple matches")
			}
		})
	}
}

func TestNormalizeCommandName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{":help", "/help"},
		{":quit", "/quit"},
		{"/help", "/help"},
		{"/quit", "/quit"},
		{"", ""},
		{":new", "/new"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeCommandName(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
