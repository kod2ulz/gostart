package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	json "github.com/json-iterator/go"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/collections"
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

func (c *client[T]) Request(ctx context.Context, method, path string) (out api.Response[T]) {
	var err api.Error
	c.start = time.Now()
	_url, parseErr := url.Parse(c.url(path))
	defer c.logOutcome(_url.String(), method, err)
	if parseErr != nil {
		err = api.RequestLoadError[T](parseErr).WithMessage("failed to parse url")
		return api.ErrorResponse[T](err)
	}
	c.setOverrides(ctx)
	setUrlQueryParams(_url, c.params)
	req, reqErr := newHttpRequest(_url, method, c.body)
	if reqErr != nil {
		err = api.RequestLoadError[T](parseErr).WithMessage("failed to create http request")
		return api.ErrorResponse[T](err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var httpClient http.Client = *http.DefaultClient
	res, resErr := httpClient.Do(req.WithContext(reqCtx))
	if resErr != nil {
		err = api.ServerError(errors.Wrap(resErr, "request failed"))
		return api.ErrorResponse[T](err)
	}
	defer res.Body.Close()
	if out = c.getResponse(res); out.HasError() {
		c.logOutcome(_url.String(), method, out.Error)
	}
	return 
}

func (c *client[T]) url(path string) string {
	return strings.Join(collections.ListReduce([]string{c.baseUrl, path}, func(_ int, s string) (string, bool) {
		elem := strings.Trim(s, " /")
		return elem, elem != ""
	}), "/")
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

func (c *client[T]) logOutcome(url, method string, err api.Error) {
	fields := logrus.Fields{
		"url":     url,
		"method":  method,
		"latency": time.Since(c.start).Milliseconds(),
	}
	if !c.headers.Empty() && c.headers.HasKey(api.RequestID) {
		fields["request_id"] = c.headers[api.RequestID]
	}
	if !c.params.Empty() {
		fields["params"] = c.params
	}
	if err == nil {
		c.log.WithFields(fields).Info()
	} else if er, ok := err.(*api.ErrorModel[T]); ok {
		fields["httpCode"] = er.Http
		c.log.WithFields(fields).WithError(err).Error(er.Code)
	}
}

func (c *client[T]) getResponse(res *http.Response) (out api.Response[T]) {
	if res.ContentLength == 0 {
		return
	}
	data, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return api.ErrorResponse[T](api.GeneralError[T](errors.Wrap(readErr, "failed to read json body into []byte")).
			WithErrorCode(api.ErrorCodeResponseProcessingError))
	}
	out = api.Response[T]{}
	var t *T = new(T)
	var errBody collections.Map[string, interface{}]
	resErr := api.GeneralError[T](errors.New(res.Status + ". request failed")).
		WithHttpStatusCode(res.StatusCode).WithErrorCode(api.ErrorCodeServiceError).
		WithError(errors.Errorf("call to %s returned %d: %s", res.Request.RequestURI, res.StatusCode, res.Status))
	if unmarshallErr := json.Unmarshal(data, &out); unmarshallErr == nil && !reflect.DeepEqual(out, api.Response[T]{}) {
		if out.HasError() {
			out.Error = resErr.WithCause(out.Error)
		} else {
			out.Success = res.StatusCode < 400
		}
		out.Timestamp = time.Now().Unix()
		return
	} else if unmarshallErr := json.Unmarshal(data, &t); unmarshallErr == nil && t != nil {
		out = api.DataResponse[T](*t)
		// anything else
	} else if unmarshallErr = json.Unmarshal(data, &errBody); unmarshallErr != nil {
		if res.StatusCode < 400 {
			t = new(T)
			return api.DataResponse[T](*t)
		} else if errorMessage := errBody.AnyOfKey("err", "error", "errors", "msg", "message"); errorMessage != nil && errorMessage != "" {
			return api.ErrorResponse[T](resErr.WithCause(api.GeneralError[any](errors.New(fmt.Sprint(errorMessage)))))
		}
	} else if errBody.Empty() {
		t = new(T)
		return api.DataResponse[T](*t)
	}

	return
}
