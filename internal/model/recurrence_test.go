package model_test

import (
	"testing"
	"time"

	"example.com/taskservice/internal/model"
)

// helper: parse "YYYY-MM-DD" → time.Time (panics on bad input, safe in tests).
func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// formatDates converts []time.Time → []string for easy assertion.
func formatDates(ts []time.Time) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Format("2006-01-02")
	}
	return out
}

// ── Validate ──────────────────────────────────────────────────────────────────

func TestRecurrenceValidate(t *testing.T) {
	interval1 := 1
	day15 := 15
	day31 := 31
	parityEven := model.ParityEven

	tests := []struct {
		name    string
		rec     model.Recurrence
		wantErr bool
	}{
		{
			name:    "daily valid",
			rec:     model.Recurrence{Type: model.RecurrenceDaily, Interval: &interval1},
			wantErr: false,
		},
		{
			name:    "daily missing interval",
			rec:     model.Recurrence{Type: model.RecurrenceDaily},
			wantErr: true,
		},
		{
			name:    "daily interval zero",
			rec:     model.Recurrence{Type: model.RecurrenceDaily, Interval: ptr(0)},
			wantErr: true,
		},
		{
			name:    "monthly valid",
			rec:     model.Recurrence{Type: model.RecurrenceMonthly, DayOfMonth: &day15},
			wantErr: false,
		},
		{
			name:    "monthly day 31 invalid",
			rec:     model.Recurrence{Type: model.RecurrenceMonthly, DayOfMonth: &day31},
			wantErr: true,
		},
		{
			name:    "monthly missing day",
			rec:     model.Recurrence{Type: model.RecurrenceMonthly},
			wantErr: true,
		},
		{
			name:    "specific_dates valid",
			rec:     model.Recurrence{Type: model.RecurrenceSpecificDates, Dates: []string{"2024-01-01"}},
			wantErr: false,
		},
		{
			name:    "specific_dates empty",
			rec:     model.Recurrence{Type: model.RecurrenceSpecificDates, Dates: nil},
			wantErr: true,
		},
		{
			name:    "specific_dates bad format",
			rec:     model.Recurrence{Type: model.RecurrenceSpecificDates, Dates: []string{"01-01-2024"}},
			wantErr: true,
		},
		{
			name:    "even_odd valid",
			rec:     model.Recurrence{Type: model.RecurrenceEvenOdd, Parity: &parityEven},
			wantErr: false,
		},
		{
			name:    "even_odd missing parity",
			rec:     model.Recurrence{Type: model.RecurrenceEvenOdd},
			wantErr: true,
		},
		{
			name:    "unknown type",
			rec:     model.Recurrence{Type: "weekly"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ── Daily ─────────────────────────────────────────────────────────────────────

func TestOccurrences_Daily(t *testing.T) {
	t.Run("every day", func(t *testing.T) {
		r := model.Recurrence{Type: model.RecurrenceDaily, Interval: ptr(1)}
		anchor := date("2024-01-01")
		got := formatDates(r.Occurrences(anchor, date("2024-01-03"), date("2024-01-05")))
		want := []string{"2024-01-03", "2024-01-04", "2024-01-05"}
		assertSliceEqual(t, want, got)
	})

	t.Run("every 2 days anchored", func(t *testing.T) {
		// anchor=Jan1, interval=2 → Jan1,Jan3,Jan5,Jan7…
		r := model.Recurrence{Type: model.RecurrenceDaily, Interval: ptr(2)}
		anchor := date("2024-01-01")
		got := formatDates(r.Occurrences(anchor, date("2024-01-01"), date("2024-01-09")))
		want := []string{"2024-01-01", "2024-01-03", "2024-01-05", "2024-01-07", "2024-01-09"}
		assertSliceEqual(t, want, got)
	})

	t.Run("every 3 days, from lands between occurrences", func(t *testing.T) {
		// anchor=Jan1, interval=3 → Jan1,Jan4,Jan7,Jan10…
		// from=Jan2 → first hit is Jan4
		r := model.Recurrence{Type: model.RecurrenceDaily, Interval: ptr(3)}
		anchor := date("2024-01-01")
		got := formatDates(r.Occurrences(anchor, date("2024-01-02"), date("2024-01-10")))
		want := []string{"2024-01-04", "2024-01-07", "2024-01-10"}
		assertSliceEqual(t, want, got)
	})

	t.Run("anchor after to — empty", func(t *testing.T) {
		r := model.Recurrence{Type: model.RecurrenceDaily, Interval: ptr(1)}
		got := r.Occurrences(date("2025-01-01"), date("2024-01-01"), date("2024-01-05"))
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("from equals to", func(t *testing.T) {
		r := model.Recurrence{Type: model.RecurrenceDaily, Interval: ptr(1)}
		anchor := date("2024-01-01")
		got := formatDates(r.Occurrences(anchor, date("2024-01-03"), date("2024-01-03")))
		want := []string{"2024-01-03"}
		assertSliceEqual(t, want, got)
	})
}

// ── Monthly ───────────────────────────────────────────────────────────────────

func TestOccurrences_Monthly(t *testing.T) {
	t.Run("15th of each month", func(t *testing.T) {
		r := model.Recurrence{Type: model.RecurrenceMonthly, DayOfMonth: ptr(15)}
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-01-01"), date("2024-04-30")))
		want := []string{"2024-01-15", "2024-02-15", "2024-03-15", "2024-04-15"}
		assertSliceEqual(t, want, got)
	})

	t.Run("day 30 — February skipped", func(t *testing.T) {
		r := model.Recurrence{Type: model.RecurrenceMonthly, DayOfMonth: ptr(30)}
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-01-01"), date("2024-04-30")))
		// Feb 2024 has 29 days (leap year), so day 30 is skipped.
		want := []string{"2024-01-30", "2024-03-30", "2024-04-30"}
		assertSliceEqual(t, want, got)
	})

	t.Run("day 29 in non-leap February skipped", func(t *testing.T) {
		r := model.Recurrence{Type: model.RecurrenceMonthly, DayOfMonth: ptr(29)}
		// 2023 is not a leap year.
		got := formatDates(r.Occurrences(date("2023-01-01"), date("2023-01-01"), date("2023-04-30")))
		want := []string{"2023-01-29", "2023-03-29", "2023-04-29"}
		assertSliceEqual(t, want, got)
	})

	t.Run("from after the day in first month", func(t *testing.T) {
		r := model.Recurrence{Type: model.RecurrenceMonthly, DayOfMonth: ptr(10)}
		// from=Jan 15 → Jan 10 already passed, first hit is Feb 10.
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-01-15"), date("2024-03-31")))
		want := []string{"2024-02-10", "2024-03-10"}
		assertSliceEqual(t, want, got)
	})
}

// ── SpecificDates ─────────────────────────────────────────────────────────────

func TestOccurrences_SpecificDates(t *testing.T) {
	t.Run("all in range", func(t *testing.T) {
		r := model.Recurrence{
			Type:  model.RecurrenceSpecificDates,
			Dates: []string{"2024-03-08", "2024-06-12", "2024-12-31"},
		}
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-01-01"), date("2024-12-31")))
		want := []string{"2024-03-08", "2024-06-12", "2024-12-31"}
		assertSliceEqual(t, want, got)
	})

	t.Run("some outside range", func(t *testing.T) {
		r := model.Recurrence{
			Type:  model.RecurrenceSpecificDates,
			Dates: []string{"2024-01-01", "2024-06-01", "2025-01-01"},
		}
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-03-01"), date("2024-12-31")))
		want := []string{"2024-06-01"}
		assertSliceEqual(t, want, got)
	})

	t.Run("unordered input — output is sorted", func(t *testing.T) {
		r := model.Recurrence{
			Type:  model.RecurrenceSpecificDates,
			Dates: []string{"2024-05-10", "2024-02-14", "2024-09-01"},
		}
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-01-01"), date("2024-12-31")))
		want := []string{"2024-02-14", "2024-05-10", "2024-09-01"}
		assertSliceEqual(t, want, got)
	})

	t.Run("none in range", func(t *testing.T) {
		r := model.Recurrence{
			Type:  model.RecurrenceSpecificDates,
			Dates: []string{"2023-01-01"},
		}
		got := r.Occurrences(date("2024-01-01"), date("2024-01-01"), date("2024-12-31"))
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})
}

// ── EvenOdd ───────────────────────────────────────────────────────────────────

func TestOccurrences_EvenOdd(t *testing.T) {
	t.Run("even days in January 2024 first week", func(t *testing.T) {
		p := model.ParityEven
		r := model.Recurrence{Type: model.RecurrenceEvenOdd, Parity: &p}
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-01-01"), date("2024-01-07")))
		want := []string{"2024-01-02", "2024-01-04", "2024-01-06"}
		assertSliceEqual(t, want, got)
	})

	t.Run("odd days in January 2024 first week", func(t *testing.T) {
		p := model.ParityOdd
		r := model.Recurrence{Type: model.RecurrenceEvenOdd, Parity: &p}
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-01-01"), date("2024-01-07")))
		want := []string{"2024-01-01", "2024-01-03", "2024-01-05", "2024-01-07"}
		assertSliceEqual(t, want, got)
	})

	t.Run("from equals to on odd day with odd parity", func(t *testing.T) {
		p := model.ParityOdd
		r := model.Recurrence{Type: model.RecurrenceEvenOdd, Parity: &p}
		got := formatDates(r.Occurrences(date("2024-01-01"), date("2024-01-03"), date("2024-01-03")))
		want := []string{"2024-01-03"}
		assertSliceEqual(t, want, got)
	})

	t.Run("from > to — empty", func(t *testing.T) {
		p := model.ParityEven
		r := model.Recurrence{Type: model.RecurrenceEvenOdd, Parity: &p}
		got := r.Occurrences(date("2024-01-01"), date("2024-01-10"), date("2024-01-01"))
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func assertSliceEqual(t *testing.T, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("length mismatch: want %d %v, got %d %v", len(want), want, len(got), got)
		return
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("index %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func ptr[T any](v T) *T { return &v }
