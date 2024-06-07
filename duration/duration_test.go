package duration

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  time.Duration
		tolerance time.Duration
		wantErr   bool
	}{
		{name: "seconds", input: "30s", expected: 30 * time.Second, tolerance: 0, wantErr: false},
		{name: "seconds", input: "-30s", expected: -30 * time.Second, tolerance: 0, wantErr: false},
		{name: "minutes", input: "15m", expected: 15 * time.Minute, tolerance: 0, wantErr: false},
		{name: "hours", input: "2h", expected: 2 * time.Hour, tolerance: 0, wantErr: false},
		{name: "days", input: "7d", expected: 7 * 24 * time.Hour, tolerance: 0, wantErr: false},
		{name: "weeks", input: "2w", expected: 2 * 7 * 24 * time.Hour, tolerance: 0, wantErr: false},
		{name: "months", input: "1M", expected: 30 * 24 * time.Hour, tolerance: 0, wantErr: false},
		{name: "years", input: "1y", expected: 365 * 24 * time.Hour, tolerance: 0, wantErr: false},
		{name: "decimal hours", input: "2.3h", expected: 2*time.Hour + 18*time.Minute, tolerance: time.Second, wantErr: false},
		{name: "decimal hours", input: "-2.3h", expected: -(2*time.Hour + 18*time.Minute), tolerance: time.Second, wantErr: false},
		{name: "invalid unit", input: "10x", wantErr: true},
		{name: "invalid value", input: "abc10s", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotRounded := got.Round(tt.tolerance)
				expectedRounded := tt.expected.Round(tt.tolerance)
				if gotRounded != expectedRounded {
					t.Errorf("ParseDuration() = %v, want %v (tolerance: %v)", got, tt.expected, tt.tolerance)
				}
			}
		})
	}
}

func TestRelativeAge(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{name: "seconds", input: 30 * time.Second, expected: "30s"},
		{name: "minutes", input: 15 * time.Minute, expected: "15m"},
		{name: "hours", input: 2 * time.Hour, expected: "2h"},
		{name: "days", input: 3 * 24 * time.Hour, expected: "3d"},
		{name: "weeks", input: 2 * 7 * 24 * time.Hour, expected: "2w"},
		{name: "months", input: 45 * 24 * time.Hour, expected: "1M"},
		{name: "years", input: 400 * 24 * time.Hour, expected: "1y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RelativeAge(tt.input); got != tt.expected {
				t.Errorf("RelativeAge() = %v, want %v", got, tt.expected)
			}
		})
	}
}
