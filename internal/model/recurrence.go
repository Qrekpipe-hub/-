package model

import (
	"fmt"
	"sort"
	"time"
)

// RecurrenceType identifies which recurrence rule is used.
type RecurrenceType string

const (
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthly       RecurrenceType = "monthly"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceEvenOdd       RecurrenceType = "even_odd"
)

// Parity is used with RecurrenceEvenOdd.
type Parity string

const (
	ParityEven Parity = "even"
	ParityOdd  Parity = "odd"
)

// Recurrence defines when a periodic task should repeat.
//
// Only the fields relevant to the chosen Type are required:
//
//	daily          → Interval  (every N days, N >= 1)
//	monthly        → DayOfMonth (1-30; months shorter than this day are skipped)
//	specific_dates → Dates     (non-empty slice of "YYYY-MM-DD" strings)
//	even_odd       → Parity    ("even" or "odd" day-of-month)
type Recurrence struct {
	Type       RecurrenceType `json:"type"`
	Interval   *int           `json:"interval,omitempty"`
	DayOfMonth *int           `json:"day_of_month,omitempty"`
	Dates      []string       `json:"dates,omitempty"`
	Parity     *Parity        `json:"parity,omitempty"`
}

// Validate checks that the rule is self-consistent.
func (r *Recurrence) Validate() error {
	if r == nil {
		return nil
	}
	switch r.Type {
	case RecurrenceDaily:
		if r.Interval == nil || *r.Interval < 1 {
			return fmt.Errorf("daily recurrence requires interval >= 1")
		}
	case RecurrenceMonthly:
		if r.DayOfMonth == nil || *r.DayOfMonth < 1 || *r.DayOfMonth > 30 {
			return fmt.Errorf("monthly recurrence requires day_of_month between 1 and 30")
		}
	case RecurrenceSpecificDates:
		if len(r.Dates) == 0 {
			return fmt.Errorf("specific_dates recurrence requires at least one date")
		}
		for _, d := range r.Dates {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", d)
			}
		}
	case RecurrenceEvenOdd:
		if r.Parity == nil {
			return fmt.Errorf("even_odd recurrence requires parity field")
		}
		if *r.Parity != ParityEven && *r.Parity != ParityOdd {
			return fmt.Errorf("parity must be %q or %q", ParityEven, ParityOdd)
		}
	default:
		return fmt.Errorf("unknown recurrence type %q", r.Type)
	}
	return nil
}

// Occurrences returns all dates in the closed interval [from, to] on which
// the rule fires. The anchor parameter is used only by RecurrenceDaily: it is
// the date the sequence starts from (normally the parent task's due_date) so
// that "every 3 days" is measured from the original task, not from `from`.
//
// All returned times are midnight UTC.
func (r *Recurrence) Occurrences(anchor, from, to time.Time) []time.Time {
	from = truncateDay(from)
	to = truncateDay(to)
	anchor = truncateDay(anchor)

	if from.After(to) {
		return nil
	}

	var result []time.Time

	switch r.Type {

	case RecurrenceDaily:
		interval := *r.Interval
		// Start the sequence at anchor; advance until we reach `from`.
		cur := anchor
		if cur.After(to) {
			return nil
		}
		// Fast-forward to the first occurrence >= from.
		if cur.Before(from) {
			diff := int(from.Sub(cur).Hours()/24) + 1
			steps := (diff + interval - 1) / interval // ceiling division
			cur = cur.AddDate(0, 0, steps*interval)
		}
		for !cur.After(to) {
			result = append(result, cur)
			cur = cur.AddDate(0, 0, interval)
		}

	case RecurrenceMonthly:
		day := *r.DayOfMonth
		// Walk month by month through the range.
		cur := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		for !cur.After(to) {
			dim := daysInMonth(cur.Year(), cur.Month())
			if day <= dim {
				candidate := time.Date(cur.Year(), cur.Month(), day, 0, 0, 0, 0, time.UTC)
				if !candidate.Before(from) && !candidate.After(to) {
					result = append(result, candidate)
				}
			}
			// If day > daysInMonth the month is intentionally skipped
			// (e.g. day=30 skips February entirely).
			cur = cur.AddDate(0, 1, 0)
		}

	case RecurrenceSpecificDates:
		for _, d := range r.Dates {
			t, _ := time.Parse("2006-01-02", d)
			t = truncateDay(t)
			if !t.Before(from) && !t.After(to) {
				result = append(result, t)
			}
		}
		sort.Slice(result, func(i, j int) bool { return result[i].Before(result[j]) })

	case RecurrenceEvenOdd:
		cur := from
		for !cur.After(to) {
			d := cur.Day()
			include := (*r.Parity == ParityEven && d%2 == 0) ||
				(*r.Parity == ParityOdd && d%2 != 0)
			if include {
				result = append(result, cur)
			}
			cur = cur.AddDate(0, 0, 1)
		}
	}

	return result
}

// ── helpers ──────────────────────────────────────────────────────────────────

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func daysInMonth(year int, month time.Month) int {
	// day 0 of the next month == last day of this month
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
