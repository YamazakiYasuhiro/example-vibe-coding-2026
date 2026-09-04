package main

import (
	"strings"
	"testing"
	"time"
)

func TestFormatToday(t *testing.T) {
	base := time.Date(2026, 9, 4, 15, 30, 0, 0, time.Local)

	tests := []struct {
		name    string
		format  string
		want    string
		wantErr bool
	}{
		{name: "iso", format: "iso", want: "2026-09-04", wantErr: false},
		{name: "slash", format: "slash", want: "2026/09/04", wantErr: false},
		{name: "jp", format: "jp", want: "2026年09月04日", wantErr: false},
		{name: "empty_as_iso", format: "", want: "2026-09-04", wantErr: false},
		{name: "unknown", format: "unknown", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatToday(base, tt.format)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("FormatToday(%q) error = nil, want error", tt.format)
				}
				if !strings.Contains(err.Error(), tt.format) {
					t.Fatalf("FormatToday(%q) error = %q, want substring %q", tt.format, err.Error(), tt.format)
				}
				return
			}
			if err != nil {
				t.Fatalf("FormatToday(%q) unexpected error: %v", tt.format, err)
			}
			if got != tt.want {
				t.Fatalf("FormatToday(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}
