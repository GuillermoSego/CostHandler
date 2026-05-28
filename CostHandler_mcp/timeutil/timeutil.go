package timeutil

import (
	"log"
	"os"
	"time"
)

var loc *time.Location

func init() {
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "America/Mexico_City"
	}

	var err error
	loc, err = time.LoadLocation(tz)
	if err != nil {
		log.Fatalf("timeutil: zona horaria inválida %q: %v", tz, err)
	}
}

func Now() time.Time {
	return time.Now().In(loc)
}

func Location() *time.Location {
	return loc
}
