package api

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	RequestID = "X-REQUEST-ID"
)

type Session interface {
	Authorization() string
}

func InitHttpClient(log *logrus.Entry, conf *HttpClientConfig, session Session) (client *HttpClient) {
	client = &HttpClient{
		Logger:         log,
		session:        session,
		client:         *http.DefaultClient,
		retryTimes:     conf.RetryTimes,
		retryTimeout:   conf.RetryTimeout,
		requestTimeout: conf.RequestTimeout,
	}

	return
}

type HttpClient struct {
	retryTimes     int
	retryTimeout   time.Duration
	requestTimeout time.Duration

	client  http.Client
	session Session

	Logger *logrus.Entry
}

func (c *HttpClient) Request(ctx context.Context) *httpClientRequest {
	return &httpClientRequest{
		log:     c.Logger,
		ctx:     ctx,
		http:    c,
		params:  Params{},
		start:   time.Now(),
		timeout: c.requestTimeout,
		headers: Headers{
			"Content-Type": "application/json",
		},
	}
}

func (c *HttpClient) WithTimeout(timeout time.Duration) *HttpClient {
	c.requestTimeout = timeout
	return c
}
