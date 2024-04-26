package utils

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Value string

func (v Value) Int() int {
	return String.ToInt(strings.Trim(string(v), " "))
}

func (v Value) Int32() int32 {
	return int32(v.Int())
}

func (v Value) Int64() int64 {
	return int64(v.Int())
}

func (v Value) Fload32() (float32) {
	v_flt, _ := strconv.ParseFloat(v.String(), 32)
	return float32(v_flt)
}

func (v Value) Float64() float64 {
	v_flt, _ := strconv.ParseFloat(v.String(), 64)
	return v_flt
}

func (v Value) String() string {
	return string(v)
}

func (v Value) StringList(separator ...string) []string {
	sep := " "
	if len(separator) > 0 {
		sep = strings.Join(separator, "")
	}
	return strings.Split(string(v), sep)
}

func (v Value) UUID() uuid.UUID {
	return uuid.MustParse(v.String())
}

func (v Value) Bool() (b bool) {
	var e error
	if b, e = strconv.ParseBool(v.String()); e != nil {
		log.WithError(e).Errorf("error parsing %s", v.String())
	}
	return
}

func (v Value) Duration() time.Duration {
	d, e := time.ParseDuration(v.String())
	if e != nil {
		log.Errorf("Error parsing duration '%v'. %v", v, e)
	}
	return d
}

func (v Value) Time(layout string) time.Time {
	t, e := time.Parse(layout, v.String())
	if e != nil {
		log.Errorf("Error parsing time '%v'. %v", v, e)
	}
	return t
}

func (v Value) Location() (loc *time.Location) {
	var err error
	loc, err = time.LoadLocation(v.String())

	if err != nil {
		loc = time.UTC
	}
	return
}

func (v Value) Valid() bool {
	return strings.Trim(string(v), " ") != ""
}
