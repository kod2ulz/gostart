package http

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ResponseHandler[T any] func(resp *http.Response) (T, error)

func LogSimpleGetRequest[T any](ctx context.Context, log *logr.Logger, url string, onResponse ResponseHandler[T]) (out T, err error) {
	start := time.Now()
	var resp *http.Response = &http.Response{}
	defer func(st time.Time) {
		fields, msg := logrus.Fields{
			"url":      url,
			"method":   http.MethodGet,
			"latency":  time.Since(st).Milliseconds(),
		}, ""
		if resp != nil {
			fields["size"] = resp.ContentLength
			fields["status"] = resp.StatusCode
			msg = resp.Status
		}
		log.WithFields(fields).Info(msg)
	}(start)
	if resp, err = http.Get(url); err != nil {
		return out, errors.Wrapf(err, "error fetching from %s", url)
	} else if resp.StatusCode != 200 {
		return out, errors.Errorf("failed with %d: %s", resp.StatusCode, resp.Status)
	} else if resp.ContentLength > 0 {
		defer resp.Body.Close()
	}
	return onResponse(resp)
}

type Payload struct {
	body io.Reader
	params map[string][]string
}

func (p *Payload) SetParams(params map[string][]string) *Payload {
	p.params = params
	return p
}

func (p *Payload) SetBody(body io.Reader) *Payload {
	p.body = body
	return p
}

func WithBodyPayload(body io.Reader) *Payload {
	return &Payload{body: body}
}

func WithParamsPayload(params map[string][]string) *Payload {
	return &Payload{params: params}
}