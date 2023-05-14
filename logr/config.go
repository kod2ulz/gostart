package logr

import (
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	ENV_LOG_LEVEL  = "LOG_LEVEL"
	ENV_PRETTY_LOG = "PRETTY_LOG_PRINT"
	ENV_APP_NAME   = "APP_NAME"
	ENV_APP_HOST   = "HOST"
)

func Config() (err error) {
	log := logrus.New()
	log.SetReportCaller(true)
	log.SetLevel(logrus.TraceLevel)

	var prettyPrint bool
	if prettyPrint, err = strconv.ParseBool(os.Getenv(ENV_PRETTY_LOG)); err != nil {
		log.WithError(err).Errorf("error parsing %s", ENV_PRETTY_LOG)
		prettyPrint = true
		log.Infof("defaulting to log.Formatter.PrettyPrint=%v", prettyPrint)
	}

	log.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint: prettyPrint,
	})

	defaultFields := logrus.Fields{
		"application": os.Getenv(ENV_APP_NAME),
	}

	if err = SetUpLogger(log.WithFields(defaultFields)); err != nil {
		log.WithError(err).WithField("logrus", map[string]interface{}{
			"level": log.GetLevel(),
		}).Error("Error setting up logger")
	}
	return
}


func _getLogLevel() logrus.Level {
	switch level := strings.ToLower(os.Getenv(ENV_LOG_LEVEL)); level {
	case "panic", "0":
		return logrus.PanicLevel
	case "fatal", "1":
		return logrus.FatalLevel
	case "error", "2":
		return logrus.ErrorLevel
	case "warn", "3":
		return logrus.WarnLevel
	case "info", "4":
		return logrus.InfoLevel
	case "debug", "5":
		return logrus.DebugLevel
	case "trace", "6":
		return logrus.TraceLevel
	}
	return logrus.TraceLevel
}

func _getHost() string {
	host, err := os.Hostname()
	if err == nil {
		return host
	}
	return os.Getenv(ENV_APP_HOST)
}