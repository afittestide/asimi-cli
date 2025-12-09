package main

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{
			name:    "version with v prefix",
			version: "v0.1.0",
			wantErr: false,
		},
		{
			name:    "version without v prefix",
			version: "0.1.0",
			wantErr: false,
		},
		{
			name:    "invalid version",
			version: "invalid",
			wantErr: true,
		},
		{
			name:    "empty version",
			version: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetUpdateCommand(t *testing.T) {
	cmd := GetUpdateCommand()
	if cmd == "" {
		t.Error("GetUpdateCommand() returned empty string")
	}
	// Should return either brew command or asimi update
	if cmd != "brew upgrade asimi" && cmd != "asimi update" {
		t.Errorf("GetUpdateCommand() = %v, want 'brew upgrade asimi' or 'asimi update'", cmd)
	}
}

func TestAutoCheckForUpdates(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "dev version skips check",
			version: "dev",
			want:    false,
		},
		{
			name:    "empty version skips check",
			version: "",
			want:    false,
		},
		// Note: We can't test actual update checking without hitting GitHub API
		// which would be flaky in tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoCheckForUpdates(tt.version)
			if got != tt.want {
				t.Errorf("AutoCheckForUpdates() = %v, want %v", got, tt.want)
			}
		})
	}
}
