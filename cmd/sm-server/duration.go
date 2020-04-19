package main

// Adapted from http://github.com/BrianHicks/finch

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"text/template"
	"time"
)

const (
	day = 24 * int64(time.Hour)
        month = int64(30.6 * float32(day))
	week = 7 * day
        year = 365 * day
)

var (
	// ErrBadFormat is returned when parsing fails
	ErrBadFormat = errors.New("bad format string")

	tmpl = template.Must(template.New("duration").Parse(`P{{if .Years}}{{.Years}}Y{{end}}{{if .Months}}{{.Months}}M{{end}}{{if .Weeks}}{{.Weeks}}W{{end}}{{if .Days}}{{.Days}}D{{end}}{{if .HasTimePart}}T{{end }}{{if .Hours}}{{.Hours}}H{{end}}{{if .Minutes}}{{.Minutes}}M{{end}}{{if .Seconds}}{{.Seconds}}S{{end}}`))

	full = regexp.MustCompile(`P((?P<year>\d+)Y)?((?P<month>\d+)M)?((?P<day>\d+)D)?(T((?P<hour>\d+)H)?((?P<minute>\d+)M)?((?P<second>\d+)S)?)?`)
	weekExpression = regexp.MustCompile(`P((?P<week>\d+)W)`)
)

type Duration struct {
	Years   int64
	Months  int64
	Weeks   int64
	Days    int64
	Hours   int64
	Minutes int64
	Seconds int64
}

func FromString(dur string) (*Duration, error) {
	var (
		match []string
		re    *regexp.Regexp
	)

	if weekExpression.MatchString(dur) {
		match = weekExpression.FindStringSubmatch(dur)
		re = weekExpression
	} else if full.MatchString(dur) {
		match = full.FindStringSubmatch(dur)
		re = full
	} else {
		return nil, ErrBadFormat
	}

	d := &Duration{}

	for i, name := range re.SubexpNames() {
		part := match[i]
		if i == 0 || name == "" || part == "" {
			continue
		}

		val, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, err
		}
		switch name {
		case "year":
			d.Years = val
		case "month":
			d.Months = val
		case "week":
			d.Weeks = val
		case "day":
			d.Days = val
		case "hour":
			d.Hours = val
		case "minute":
			d.Minutes = val
		case "second":
			d.Seconds = val
		default:
			return nil, errors.New(fmt.Sprintf("unknown field %s", name))
		}
	}

	return d, nil
}

// String prints out the value passed in. It's not strictly according to the
// ISO spec, but it's pretty close. In particular, to completely conform it
// would need to round up to the next largest unit. 61 seconds to 1 minute 1
// second, for example. It would also need to disallow weeks mingling with
// other units.
func (d *Duration) String() string {
	var s bytes.Buffer

	err := tmpl.Execute(&s, d)
	if err != nil {
		panic(err)
	}

	return s.String()
}

func (d *Duration) HasTimePart() bool {
	return d.Hours != 0 || d.Minutes != 0 || d.Seconds != 0
}

func (d *Duration) ToDuration() time.Duration {
	var tot int64 = 0

	tot += year * d.Years
	tot += month * d.Months
	tot += week * d.Weeks
	tot += day * d.Days
	tot += int64(time.Hour) * d.Hours
	tot += int64(time.Minute) * d.Minutes
	tot += int64(time.Second) * d.Seconds

	return time.Duration(tot)
}
