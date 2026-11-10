// Package iso_8601_duration_parser converts between time.Duration and ISO 8601
// duration strings such as P1Y2M3DT4H5M6S.
//
// The conversion is lossy in both directions because ISO 8601 durations are
// calendar-based (years, months) while time.Duration is a fixed number of
// nanoseconds. We resolve the ambiguity by fixing the length of a year to 365
// days and a month to 30 days. This is a documented, deterministic choice:
// callers who need calendar-accurate arithmetic should use a time.Location and
// AddDate instead.
package iso_8601_duration_parser

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Fixed lengths used when converting between calendar units and time.Duration.
// These are deliberately not configurable: a single, documented convention is
// more useful than a knob that silently changes serialised values.
const (
	daysPerYear  = 365
	daysPerMonth = 30
)

// Duration is the result of parsing an ISO 8601 duration. It keeps the
// calendar components separate from the clock components so that callers can
// inspect what was actually written, and it exposes a time.Duration view via
// Duration().
type Duration struct {
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
	Seconds float64
}

// ErrInvalidDuration is returned when a string is not a valid ISO 8601
// duration. Use errors.Is to test for it.
var ErrInvalidDuration = errors.New("invalid ISO 8601 duration")

// durationPattern matches the subset of ISO 8601 durations we accept:
// a leading P, optional date components, optional T followed by time
// components. At least one component must be present. The designator letters
// must appear in the canonical order; a component may be omitted entirely.
var durationPattern = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)

// Parse parses an ISO 8601 duration string such as "P1Y2M3DT4H5M6S" and
// returns a Duration. The syntax accepted is the common subset:
//
//	P[nY][nM][nD][T[nH][nM][nS]]
//
// where the seconds component may have a fractional part. A bare "P" or a
// trailing "T" with no time components is rejected, as is any string with a
// component out of order or with a negative sign (ISO 8601 durations are
// non-negative; use a signed time.Duration if you need direction).
func Parse(s string) (Duration, error) {
	var d Duration
	m := durationPattern.FindStringSubmatch(s)
	if m == nil {
		return d, fmt.Errorf("%w: %q", ErrInvalidDuration, s)
	}

	// Reject the two empty-but-well-formed shapes: "P" and "PT".
	if s == "P" || s == "PT" {
		return d, fmt.Errorf("%w: %q", ErrInvalidDuration, s)
	}

	var err error
	if d.Years, err = parseOptionalInt(m[1]); err != nil {
		return Duration{}, fmt.Errorf("%w: %q: %v", ErrInvalidDuration, s, err)
	}
	if d.Months, err = parseOptionalInt(m[2]); err != nil {
		return Duration{}, fmt.Errorf("%w: %q: %v", ErrInvalidDuration, s, err)
	}
	if d.Days, err = parseOptionalInt(m[3]); err != nil {
		return Duration{}, fmt.Errorf("%w: %q: %v", ErrInvalidDuration, s, err)
	}
	if d.Hours, err = parseOptionalInt(m[4]); err != nil {
		return Duration{}, fmt.Errorf("%w: %q: %v", ErrInvalidDuration, s, err)
	}
	if d.Minutes, err = parseOptionalInt(m[5]); err != nil {
		return Duration{}, fmt.Errorf("%w: %q: %v", ErrInvalidDuration, s, err)
	}
	if m[6] != "" {
		d.Seconds, err = strconv.ParseFloat(m[6], 64)
		if err != nil {
			return Duration{}, fmt.Errorf("%w: %q: %v", ErrInvalidDuration, s, err)
		}
	}

	return d, nil
}

// parseOptionalInt converts a possibly-empty capture group to an int. An empty
// group means the component was absent and is treated as zero.
func parseOptionalInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

// Duration converts the parsed components to a time.Duration using the fixed
// year and month lengths documented on the package. Fractional seconds are
// preserved exactly as far as time.Duration allows; sub-nanosecond precision
// is truncated by the float-to-int conversion.
func (d Duration) Duration() time.Duration {
	totalDays := d.Years*daysPerYear + d.Months*daysPerMonth + d.Days
	seconds := float64(totalDays)*24*60*60 +
		float64(d.Hours)*3600 +
		float64(d.Minutes)*60 +
		d.Seconds
	return time.Duration(seconds * float64(time.Second))
}

// String renders the duration in the canonical ISO 8601 form used by this
// package: only non-zero components are emitted, in the order
// Y M D H M S, with the T separator present only when a time component is
// non-zero. A zero Duration renders as "PT0S".
func (d Duration) String() string {
	var b strings.Builder
	b.WriteByte('P')
	if d.Years != 0 {
		fmt.Fprintf(&b, "%dY", d.Years)
	}
	if d.Months != 0 {
		fmt.Fprintf(&b, "%dM", d.Months)
	}
	if d.Days != 0 {
		fmt.Fprintf(&b, "%dD", d.Days)
	}

	hasTime := d.Hours != 0 || d.Minutes != 0 || d.Seconds != 0
	if !hasTime {
		// A duration with no components would be a bare "P", which is not a
		// valid ISO 8601 duration. "PT0S" is the conventional zero form.
		if d.Years == 0 && d.Months == 0 && d.Days == 0 {
			return "PT0S"
		}
		return b.String()
	}

	b.WriteByte('T')
	if d.Hours != 0 {
		fmt.Fprintf(&b, "%dH", d.Hours)
	}
	if d.Minutes != 0 {
		fmt.Fprintf(&b, "%dM", d.Minutes)
	}
	if d.Seconds != 0 {
		fmt.Fprintf(&b, "%sS", formatSeconds(d.Seconds))
	}
	return b.String()
}

// formatSeconds renders a seconds value without a trailing ".0" for whole
// numbers, so that round-tripping integer seconds does not gain a decimal
// point. strconv.FormatFloat with 'f' and -1 gives the shortest exact
// representation, which is what we want for a value that came from parsing.
func formatSeconds(s float64) string {
	return strconv.FormatFloat(s, 'f', -1, 64)
}

// Format converts a time.Duration to an ISO 8601 duration string. Because
// time.Duration has no calendar components, the result uses only the clock
// components (H, M, S) plus days, which are unambiguous. Years and months are
// never emitted by Format; use Parse to read them back.
//
// The duration is treated as an absolute value: a negative time.Duration
// formats as its positive counterpart, since ISO 8601 durations in this
// package are non-negative.
func Format(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	totalSeconds := float64(d) / float64(time.Second)
	days := int(totalSeconds / 86400)
	totalSeconds -= float64(days) * 86400
	hours := int(totalSeconds / 3600)
	totalSeconds -= float64(hours) * 3600
	minutes := int(totalSeconds / 60)
	totalSeconds -= float64(minutes) * 60
	seconds := totalSeconds

	var b strings.Builder
	b.WriteByte('P')
	if days != 0 {
		fmt.Fprintf(&b, "%dD", days)
	}

	hasTime := hours != 0 || minutes != 0 || seconds != 0
	if !hasTime {
		if days == 0 {
			return "PT0S"
		}
		return b.String()
	}

	b.WriteByte('T')
	if hours != 0 {
		fmt.Fprintf(&b, "%dH", hours)
	}
	if minutes != 0 {
		fmt.Fprintf(&b, "%dM", minutes)
	}
	if seconds != 0 {
		fmt.Fprintf(&b, "%sS", formatSeconds(seconds))
	}
	return b.String()
}
