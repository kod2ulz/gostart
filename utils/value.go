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

func (v Value) Int64() int64 {
	return int64(v.Int())
}

func (v Value) String() string {
	return string(v)
}

func (v Value) StringList(separator...string) []string {
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

func (v Value) Valid() bool {
	return strings.Trim(string(v), " ") != ""
}