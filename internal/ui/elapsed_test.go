package ui

import (
	"testing"
	"time"
)

func TestFormatElapsed(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "<1m"},
		{-5 * time.Second, "<1m"},
		{59 * time.Second, "<1m"},
		{time.Minute, "1m"},
		{37 * time.Minute, "37m"},
		{time.Hour, "1h"},
		{time.Hour + 25*time.Minute, "1h25m"},
		{23*time.Hour + 59*time.Minute, "23h59m"},
		{24 * time.Hour, "1d"},
		{51 * time.Hour, "2d3h"},
	}
	for _, c := range cases {
		if got := FormatElapsed(c.d); got != c.want {
			t.Errorf("FormatElapsed(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
