package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/kod2ulz/gostart/object"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type httpClientRequest struct {
	start   time.Time
	err     error
	log     *logrus.Entry
	ctx     context.Context
	http    *HttpClient
	timeout time.Duration
	params  Params
	headers Headers
	req     *http.Request
	res     *http.Response
}

func (r *httpClientRequest) WithParam(key, value string) *httpClientRequest {
	r.params.Add(key, value)
	return r
}

func (r *httpClientRequest) WithParams(params map[string]interface{}) *httpClientRequest {
	r.params.Merge(params)
	return r
}

func (r *httpClientRequest) WithHeader(key, value string) *httpClientRequest {
	r.headers.Add(key, value)
	return r
}

func (r *httpClientRequest) WithHeaders(headers map[string]interface{}) *httpClientRequest {
	r.headers.Merge(headers)
	return r
}

func (r *httpClientRequest) Get(url string) error {
	return r.doRequest(http.MethodGet, url, r.headers, r.params, nil, nil)
}

func (r *httpClientRequest) GetWithResponseBody(url string, response interface{}) error {
	return r.doRequest(http.MethodGet, url, r.headers, r.params, nil, response)
}

func (r *httpClientRequest) Post(url string, body, response interface{}) error {
	return r.doRequest(http.MethodPost, url, r.headers, r.params, body, response)
}

func (r *httpClientRequest) Put(url string, body, response interface{}) error {
	return r.doRequest(http.MethodPut, url, r.headers, r.params, body, response)
}

func (r *httpClientRequest) doRequest(method, url string, headers Headers, params Params, body, out interface{}) error {
	r.req, r.err = params.Request(method, url, body)
	if r.err != nil {
		return utils.Error.Log(r.log, r.err, "failed to create request object")
	}
	defer r.logRequest(params, method, r.req.URL.String())
	headers.WithRequestID(r.ctx).WithAuthorization(r.http.session).Set(r.req)

	rctx, cancel := context.WithTimeout(r.ctx, r.timeout)
	defer cancel()
	r.res, r.err = r.http.client.Do(r.req.WithContext(rctx))
	if r.err != nil {
		return utils.Error.Log(r.log, r.err, "error fetching response from API")
	} else if r.res.StatusCode >= 400 {
		var data object.Map[string, interface{}]
		if r.err = utils.Net.ReadJson(r.res.Body, &data); r.err != nil {
			return utils.Error.Log(r.log, errors.Wrapf(r.err, "failed to read json body of response with %d", r.res.StatusCode), "")
		} else if errMsg := data.AnyOfKey("err", "error", "msg", "message"); errMsg != nil {
			return utils.Error.Log(r.log, errors.Errorf("request failed with %d. %v", r.res.StatusCode, errMsg), "")
		}
		return utils.Error.Log(r.log, errors.Errorf("response returned %d", r.res.StatusCode), "")
	} else if r.res.ContentLength == 0 {
		return nil
	}

	var data []byte
	data, r.err = io.ReadAll(r.res.Body)
	if r.err != nil {
		return utils.Error.Log(r.log, r.err, "error ready response body from API")
	}
	defer r.res.Body.Close()

	if r.err = json.Unmarshal(data, out); r.err != nil {
		return utils.Error.Log(r.log, r.err, "failed to unmarshall response to %T", out)
	}
	return r.err
}

func (r *httpClientRequest) logRequest(params Params, method, url string) {
	var code int = 500
	var status string = "request failed"
	if r.res != nil {
		code = r.res.StatusCode
		status = r.res.Status
	}

	fields := logrus.Fields{
		"url":      url,
		"method":   method,
		"response": code,
		"latency":  time.Since(r.start).Milliseconds(),
	}
	if len(params) != 0 {
		fields["params"] = params
	}
	if r.err != nil {
		r.log.WithFields(fields).WithError(r.err).Error(status)
	} else {
		r.log.WithFields(fields).Info(status)
	}
}
