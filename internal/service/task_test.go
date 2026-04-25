package service_test

import (
	"testing"

	"example.com/taskservice/internal/service"
)

func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid range",
			from:    "2024-01-01",
			to:      "2024-12-31",
			wantErr: false,
		},
		{
			name:    "same day",
			from:    "2024-06-15",
			to:      "2024-06-15",
			wantErr: false,
		},
		{
			name:    "empty from",
			from:    "",
			to:      "2024-12-31",
			wantErr: true,
		},
		{
			name:    "empty to",
			from:    "2024-01-01",
			to:      "",
			wantErr: true,
		},
		{
			name:    "to before from",
			from:    "2024-12-31",
			to:      "2024-01-01",
			wantErr: true,
		},
		{
			name:    "bad from format",
			from:    "01-01-2024",
			to:      "2024-12-31",
			wantErr: true,
		},
		{
			name:    "range exceeds 2 years",
			from:    "2020-01-01",
			to:      "2025-01-02",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateDateRange(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDateRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
