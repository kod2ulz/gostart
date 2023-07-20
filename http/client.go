package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func Client[T any](log *logrus.Entry) *client[T] {
	return &client[T]{
		log:     log,
		timeout: time.Minute,
		params:  map[string][]string{},
		headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

type client[T any] struct {
	baseUrl string
	start   time.Time
	out     *T
	log     *logrus.Entry
	body    any
	session Session
	timeout time.Duration
	params  collections.Map[string, []string]
	headers collections.Map[string, string]
}

func (c *client[T]) Timeout(timeut time.Duration) *client[T] {
	c.timeout = timeut
	return c
}

func (c *client[T]) BaseUrl(url string) *client[T] {
	c.baseUrl = url
	return c
}

func (c *client[T]) Body(body any) *client[T] {
	c.body = body
	return c
}

func (c *client[T]) Out(out *T) *client[T] {
	c.out = out
	return c
}

func (c *client[T]) Session(session Session) *client[T] {
	if session != nil {
		c.session = session
	}
	return c
}

func (c *client[T]) Header(key, value string) *client[T] {
	return c.Headers(map[string]string{key: value})
}

func (c *client[T]) Headers(headers Headers) *client[T] {
	c.headers.Merge(headers)
	return c
}

func (c *client[T]) Param(name, value string) *client[T] {
	return c.Params(map[string][]string{name: {value}})
}

func (c *client[T]) Params(params map[string][]string) *client[T] {
	if len(params) == 0 {
		return c
	}
	for name := range params {
		if _, ok := c.params[name]; !ok {
			c.params[name] = params[name]
		} else {
			c.params[name] = append(c.params[name], params[name]...)
		}
	}
	return c
}

func (c *client[T]) Get(ctx context.Context, path string) api.Response[T] {
	return c.Request(ctx, http.MethodGet, path)
}

func (c *client[T]) Post(ctx context.Context, path string) api.Response[T] {
	return c.Request(ctx, http.MethodPost, path)
}

func (c *client[T]) Put(ctx context.Context, path string) api.Response[T] {
	return c.Request(ctx, http.MethodPut, path)
}

func (c *client[T]) Delete(ctx context.Context, path string) api.Response[T] {
	return c.Request(ctx, http.MethodDelete, path)
}

func (c *client[T]) Request(ctx context.Context, method, path string) api.Response[T] {
	var err api.Error
	c.start = time.Now()
	_url, parseErr := url.Parse(strings.Trim(c.baseUrl+path, " /"))
	if parseErr != nil {
		return api.ErrorResponse[T](api.RequestLoadError[T](parseErr).WithMessage("failed to parse url"))
	}
	c.setOverrides(ctx)
	setUrlQueryParams(_url, c.params)
	req, reqErr := newHttpRequest(_url, method, c.body)
	if reqErr != nil {
		return api.ErrorResponse[T](api.RequestLoadError[T](parseErr).WithMessage("failed to create http request"))
	}
	defer c.logOutcome(req.URL.String(), method, err)
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var httpClient http.Client = *http.DefaultClient
	res, resErr := httpClient.Do(req.WithContext(reqCtx))
	if resErr != nil {
		err = api.ServerError(errors.Wrap(resErr, "request failed"))
		return api.ErrorResponse[T](err)
	}
	defer res.Body.Close()
	var out *T
	if out, err = c.processResponse(res); err != nil {
		return api.ErrorResponse[T](err)
	} else if out != nil {
		return api.DataResponse[T](*out)
	}
	return api.EmptyResponse[T]()
}

func (c *client[T]) setOverrides(ctx context.Context) {
	var h = Headers{}
	if ctx != nil {
		h.WithRequestID(ctx)
	}
	if c.session != nil {
		h.WithAuthorization(c.session)
	}
	c.Headers(h)
}

func (r *client[T]) logOutcome(url, method string, err api.Error) {
	fields := logrus.Fields{
		"url":     url,
		"method":  method,
		"latency": time.Since(r.start).Milliseconds(),
	}
	if !r.params.Empty() {
		fields["params"] = r.params
	}
	if err == nil {
		r.log.WithFields(fields).Info()
	} else if er, ok := err.(*api.ErrorModel[T]); ok {
		fields["httpCode"] = er.Http
		r.log.WithFields(fields).WithError(err).Error(er.Code)
	}
}

func (c *client[T]) processResponse(res *http.Response) (out *T, err api.Error) {
	if res.StatusCode >= 400 {
		err = api.GeneralError[T](errors.New(res.Status + ". request failed")).
			WithHttpStatusCode(res.StatusCode).WithErrorCode(api.ErrorCodeServiceError)
		var errBody collections.Map[string, interface{}]
		if res.ContentLength == 0 {
			return
		} else if unmarshallErr := utils.Net.ReadJson(res.Body, &errBody); unmarshallErr != nil {
			return out, api.GeneralError[T](errors.Wrap(unmarshallErr, "failed to unmarshall json")).WithCause(err)
		} else if errorMessage := errBody.AnyOfKey("err", "error", "errors", "msg", "message"); errorMessage != nil && errorMessage != "" {
			return out, err.WithError(errors.New(fmt.Sprint(errorMessage)))
		}
	} else if res.ContentLength == 0 {
		return
	} else if c.out != nil {
		out = c.out
	} else {
		out = new(T)
	}
	if unmarshallErr := utils.Net.ReadJson(res.Body, out); unmarshallErr != nil {
		return out, api.GeneralError[T](errors.Wrap(unmarshallErr, "failed to unmarshall response")).
			WithErrorCode(api.ErrorCodeResponseProcessingError)
	}
	return
}
