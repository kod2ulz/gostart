package utils

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/sirupsen/logrus"
)

type errorUtils struct{}
type logger func (string, ...any)

var Error errorUtils

func (errorUtils) Log(log *logrus.Entry, err error, message string, args ...interface{}) error {
	if err != nil {
		log.WithError(err).Errorf(message, args...)
		return err
	}
	return nil
}

func (errorUtils) LogOK(log logger, message string, args ...interface{}) error {
	if message != "" {
		log(message, args...)
	}
	return nil
}

func (errorUtils) Fail(log *logrus.Entry, err error, message string, args ...interface{}) error {
	if err != nil {
		log.WithError(err).Fatalf(message, args...)
		return err
	}
	return nil
}

func (errorUtils) SqlNoRows(err error) bool {
	return err != nil &&  errors.Is(err, sql.ErrNoRows) || strings.HasSuffix(err.Error(), "no rows in result set")
}