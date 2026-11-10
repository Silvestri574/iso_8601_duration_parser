package iso_8601_duration_parser

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestParseHappyPath(t *testing.T) {
	d, err := Parse("P1Y2M3DT4H5M6S")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	want := Duration{Years: 1, Months: 2, Days: 3, Hours: 4, Minutes: 5, Seconds: 6}
	if d != want {
		t.Fatalf("Parse = %+v, want %+v", d, want)
	}
}

func TestParseFractionalSeconds(t *testing.T) {
	d, err := Parse("PT1.5S")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if math.Abs(d.Seconds-1.5) > 1e-9 {
		t.Fatalf("Seconds = %v, want 1.5", d.Seconds)
	}
}

func TestParseMissingComponents(t *testing.T) {
	cases := []struct {
		in   string
		want Duration
	}{
		{"P1Y", Duration{Years: 1}},
		{"P1M", Duration{Months: 1}},
		{"P1D", Duration{Days: 1}},
		{"PT1H", Duration{Hours: 1}},
		{"PT1M", Duration{Minutes: 1}},
		{"PT1S", Duration{Seconds: 1}},
		{"P1DT1H", Duration{Days: 1, Hours: 1}},
	}
	for _, tc := range cases {
		d, err := Parse(tc.in)
		if err != nil {
			t.Errorf("Parse(%q) returned error: %v", tc.in, err)
			continue
		}
		if d != tc.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tc.in, d, tc.want)
		}
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	invalid := []string{
		"",
		"P",
		"PT",
		"1Y",
		"P1H",   // hours without T
		"PT1Y",  // years after T
		"P1S",   // seconds without T
		"P-1D",  // negative
		"P1.5Y", // fractional years not accepted
		"P1Y2Y", // duplicate component
		"P1Y2M3D4H",
	}
	for _, s := range invalid {
		_, err := Parse(s)
		if err == nil {
			t.Errorf("Parse(%q) succeeded, want error", s)
			continue
		}
		if !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("Parse(%q) error = %v, want ErrInvalidDuration", s, err)
		}
	}
}

func TestDurationConversion(t *testing.T) {
	d, err := Parse("P1Y2M3DT4H5M6S")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	// 1 year = 365 days, 2 months = 60 days, 3 days = 3 days.
	days := 365 + 60 + 3
	want := time.Duration(days)*24*time.Hour + 4*time.Hour + 5*time.Minute + 6*time.Second
	if got := d.Duration(); got != want {
		t.Fatalf("Duration() = %v, want %v", got, want)
	}
}

func TestDurationZero(t *testing.T) {
	d, err := Parse("PT0S")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got := d.Duration(); got != 0 {
		t.Fatalf("Duration() = %v, want 0", got)
	}
}

func TestStringRoundTrip(t *testing.T) {
	cases := []string{
		"P1Y2M3DT4H5M6S",
		"P1Y",
		"PT1H",
		"PT1.5S",
		"PT0S",
		"P1D",
	}
	for _, s := range cases {
		d, err := Parse(s)
		if err != nil {
			t.Errorf("Parse(%q) returned error: %v", s, err)
			continue
		}
		if got := d.String(); got != s {
			t.Errorf("Parse(%q).String() = %q, want %q", s, got, s)
		}
	}
}

func TestStringZero(t *testing.T) {
	if got := (Duration{}).String(); got != "PT0S" {
		t.Fatalf("zero Duration.String() = %q, want %q", got, "PT0S")
	}
}

func TestFormat(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "PT0S"},
		{time.Second, "PT1S"},
		{90 * time.Second, "PT1M30S"},
		{time.Hour, "PT1H"},
		{25 * time.Hour, "P1DT1H"},
		{1500 * time.Millisecond, "PT1.5S"},
		{-time.Second, "PT1S"},
	}
	for _, tc := range cases {
		if got := Format(tc.in); got != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	// Format never emits years or months, so parsing its output and
	// converting back to a time.Duration must be lossless for whole seconds.
	inputs := []time.Duration{
		0,
		time.Second,
		90 * time.Second,
		time.Hour,
		25 * time.Hour,
		49*time.Hour + 30*time.Minute + 15*time.Second,
	}
	for _, in := range inputs {
		s := Format(in)
		d, err := Parse(s)
		if err != nil {
			t.Errorf("Parse(Format(%v)) returned error: %v", in, err)
			continue
		}
		if got := d.Duration(); got != in {
			t.Errorf("round trip %v -> %q -> %v", in, s, got)
		}
	}
}
