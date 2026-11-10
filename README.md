# iso_8601_duration_parser

Converts between `time.Duration` and ISO 8601 duration strings such as `P1Y2M3DT4H5M6S`.

```go
package main

import (
	"fmt"
	"time"

	iso "github.com/example/iso_8601_duration_parser"
)

func main() {
	d, err := iso.Parse("P1Y2M3DT4H5M6S")
	if err != nil {
		panic(err)
	}
	fmt.Println(d.Duration()) // 399d4h5m6s
	fmt.Println(d.String())   // P1Y2M3DT4H5M6S

	fmt.Println(iso.Format(90 * time.Second)) // PT1M30S
}
```

## Why this exists

Go's standard library has no ISO 8601 duration support. `time.ParseDuration`
understands `1h30m` but not `PT1H30M`, and it has no concept of days, months, or
years. This package fills that gap for the common case of serialising and
deserialising durations in JSON, configuration files, and APIs.

The trade-off is explicit: ISO 8601 durations are calendar-based, but
`time.Duration` is a fixed count of nanoseconds. A year and a month have no
fixed length. This package fixes a year at 365 days and a month at 30 days.
That makes every conversion deterministic and round-trippable, at the cost of
being wrong for calendar arithmetic across leap years or months of different
lengths. If you need real calendar arithmetic, parse the components with
`Parse` and apply them with `time.Time.AddDate`.

## The awkward edge

`Format` never emits years or months, because a `time.Duration` cannot carry
that information. So `Format` and `Parse` are not symmetric: `Parse` can read
`P1Y`, but `Format` will only ever produce days, hours, minutes, and seconds.
Round-tripping through `Format` and then `Parse` is lossless for any duration
that is a whole number of nanoseconds, because the day boundary is exact.

A bare `P` and a trailing `T` with no time components are rejected. A zero
duration formats as `PT0S`, which is the conventional zero form.

## API

- `type Duration struct { Years, Months, Days, Hours, Minutes int; Seconds float64 }`
- `func Parse(s string) (Duration, error)`
- `func (d Duration) Duration() time.Duration`
- `func (d Duration) String() string`
- `func Format(d time.Duration) string`
- `var ErrInvalidDuration error`
