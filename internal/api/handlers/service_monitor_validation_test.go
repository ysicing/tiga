package handlers

import (
	"testing"

	"github.com/ysicing/tiga/internal/models"
)

func TestValidateAndCleanTarget(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		probeType models.ProbeType
		want      string
		wantErr   bool
	}{
		// HTTP tests
		{
			name:      "HTTP with leading/trailing spaces",
			target:    "  https://example.com  ",
			probeType: models.ProbeTypeHTTP,
			want:      "https://example.com",
			wantErr:   false,
		},
		{
			name:      "HTTP with newlines",
			target:    "https://example.com\n",
			probeType: models.ProbeTypeHTTP,
			want:      "https://example.com",
			wantErr:   false,
		},
		{
			name:      "HTTP without scheme - adds http://",
			target:    "example.com",
			probeType: models.ProbeTypeHTTP,
			want:      "http://example.com",
			wantErr:   false,
		},
		{
			name:      "HTTP with path",
			target:    "https://example.com/api/health",
			probeType: models.ProbeTypeHTTP,
			want:      "https://example.com/api/health",
			wantErr:   false,
		},
		{
			name:      "HTTP invalid URL",
			target:    "not a url",
			probeType: models.ProbeTypeHTTP,
			want:      "",
			wantErr:   true,
		},

		// ICMP tests
		{
			name:      "ICMP with IP address",
			target:    "  8.8.8.8  ",
			probeType: models.ProbeTypeICMP,
			want:      "8.8.8.8",
			wantErr:   false,
		},
		{
			name:      "ICMP with domain",
			target:    " google.com ",
			probeType: models.ProbeTypeICMP,
			want:      "google.com",
			wantErr:   false,
		},
		{
			name:      "ICMP with hostname",
			target:    "localhost",
			probeType: models.ProbeTypeICMP,
			want:      "localhost",
			wantErr:   false,
		},
		{
			name:      "ICMP with IPv6",
			target:    "2001:4860:4860::8888",
			probeType: models.ProbeTypeICMP,
			want:      "2001:4860:4860::8888",
			wantErr:   false,
		},
		{
			name:      "ICMP invalid",
			target:    "not@valid",
			probeType: models.ProbeTypeICMP,
			want:      "",
			wantErr:   true,
		},

		// TCP tests
		{
			name:      "TCP with host and port",
			target:    "  example.com:80  ",
			probeType: models.ProbeTypeTCP,
			want:      "example.com:80",
			wantErr:   false,
		},
		{
			name:      "TCP with IP and port",
			target:    "192.168.1.1:3306",
			probeType: models.ProbeTypeTCP,
			want:      "192.168.1.1:3306",
			wantErr:   false,
		},
		{
			name:      "TCP without port",
			target:    "example.com",
			probeType: models.ProbeTypeTCP,
			want:      "example.com",
			wantErr:   false,
		},
		{
			name:      "TCP with tabs",
			target:    "\texample.com:22\t",
			probeType: models.ProbeTypeTCP,
			want:      "example.com:22",
			wantErr:   false,
		},
		{
			name:      "TCP invalid port",
			target:    "example.com:99999",
			probeType: models.ProbeTypeTCP,
			want:      "",
			wantErr:   true,
		},
		{
			name:      "TCP invalid host",
			target:    "invalid@host:80",
			probeType: models.ProbeTypeTCP,
			want:      "",
			wantErr:   true,
		},

		// Edge cases
		{
			name:      "Empty target",
			target:    "   ",
			probeType: models.ProbeTypeHTTP,
			want:      "",
			wantErr:   true,
		},
		{
			name:      "Target with control characters",
			target:    "example.com\x00\x01",
			probeType: models.ProbeTypeHTTP,
			want:      "http://example.com",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateAndCleanTarget(tt.target, tt.probeType)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAndCleanTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateAndCleanTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}
