package model

import (
	"fmt"
	"sort"
	"time"
)

type RecurrenceType string

const (
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthly       RecurrenceType = "monthly"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceEvenOdd       RecurrenceType = "even_odd"
)

type Parity string

const (
	ParityEven Parity = "even"
	ParityOdd  Parity = "odd"
)

type Recurrence struct {
	Type       RecurrenceType `json:"type"`
	Interval   *int           `json:"interval,omitempty"`
	DayOfMonth *int           `json:"day_of_month,omitempty"`
	Dates      []string       `json:"dates,omitempty"`
	Parity     *Parity        `json:"parity,omitempty"`
}

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
		cur := anchor
		if cur.After(to) {
			return nil
		}
		// Fast-forward to the first occurrence >= from.
		if cur.Before(from) {
			diff := int(from.Sub(cur).Hours() / 24)
			steps := (diff + interval - 1) / interval
			cur = cur.AddDate(0, 0, steps*interval)
		}
		for !cur.After(to) {
			result = append(result, cur)
			cur = cur.AddDate(0, 0, interval)
		}

	case RecurrenceMonthly:
		day := *r.DayOfMonth
		cur := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		for !cur.After(to) {
			dim := daysInMonth(cur.Year(), cur.Month())
			if day <= dim {
				candidate := time.Date(cur.Year(), cur.Month(), day, 0, 0, 0, 0, time.UTC)
				if !candidate.Before(from) && !candidate.After(to) {
					result = append(result, candidate)
				}
			}
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

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
