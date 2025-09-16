package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/utils"
)

// RequestModal provides a comprehensive, framework-enhanced implementation for RequestParam interface.
// It extends contracts.RequestModal with framework-specific features and response handling capabilities.
// This is the recommended implementation for most use cases as it provides both request processing
// and response metadata handling in a single, convenient package.
//
// Features:
// - Framework-specific request loading (JSON body, query params, path params, headers)
// - Enhanced validation with context-aware error handling
// - Response metadata and reference management
// - Gin-specific optimizations and utilities
//
// Use this when building API handlers that need full request/response lifecycle support.
type _t struct{ RequestModal[_t] }

var _ contracts.RequestParam = RequestModal[_t]{}

type RequestModal[T contracts.RequestParam] struct {
	contracts.RequestModal[T]
}

func (r RequestModal[T]) Validate(ctx contracts.RequestContext) error {
	if ctxSetter, ok := ctx.(interface{ Value(interface{}) interface{} }); ok {
		return utils.Validate.Struct(ctxSetter.Value(r.ContextKey()))
	}
	return fmt.Errorf("cannot validate: invalid context type")
}

func (r RequestModal[T]) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
	t := new(T)
	if err = r.LoadFromJsonBody(ctx, t); err == nil {
		// Note: This is a temporary workaround - ideally we should use RequestContext interface
		if ctxSetter, ok := ctx.(interface{ Set(string, interface{}) }); ok {
			ctxSetter.Set((*t).ContextKey(), t)
		}
		return *t, err
	}
	return nil, errors.Wrapf(err, "failed to load request %T", t)
}

func (r RequestModal[T]) LoadFromJsonBody(ctx contracts.RequestContext, out interface{}) (err error) {
	if err = ctx.ShouldBindJSON(out); err != nil {
		return errors.Wrapf(err, "failed to load json body into %T from request", out)
	}
	return
}

func (r RequestModal[T]) ContextKey() string {
	var t = new(T)
	return fmt.Sprintf("%T", t)
}



func (p RequestModal[T]) ContextLoad(ctx context.Context) (out contracts.RequestParam, err error) {
	val := ctx.Value(p.ContextKey())
	if val == nil {
		return out, errors.Errorf("value of %T with key %s was %v in context", p, p.ContextKey(), val)
	}
	return val.(contracts.RequestParam), nil
}

func (p RequestModal[T]) LoadFromContext(ctx context.Context, out contracts.RequestParam) (err error) {
	var param contracts.RequestParam
	if out == nil {
		return fmt.Errorf("out is nil")
	} else if param, err = (*new(T)).ContextLoad(ctx); err != nil {
		return errors.Wrapf(err, "Failed to load %T from context", out)
	} else if param == nil {
		if param = ctx.Value(p.ContextKey()).(contracts.RequestParam); param == nil {
			return fmt.Errorf("Got %v when loading %T from context", out, out)
		}
	}
	utils.StructCopy(param, out)
	return
}

func (p RequestModal[T]) FromContext(ctx context.Context, out *T) (err error) {
	if out == nil {
		return fmt.Errorf("out is nil")
	} else if val := ctx.Value(p.ContextKey()); val == nil {
		return fmt.Errorf("value of %T with key %s was %v in context", *out, p.ContextKey(), val)
	} else if param, ok := val.(T); ok {
		*out = param
	} else {
		return fmt.Errorf("failed to cast %T to %T ", val, *out)
	}
	return
}

func (p RequestModal[T]) InContext(ctx context.Context, in T) context.Context {
	return context.WithValue(ctx, in.ContextKey(), in)
}

func (p RequestModal[T]) Query(ctx contracts.RequestContext, name string, _default ...string) (out config.Value) {
	if v := ctx.Query(name); v.Valid() {
		return config.Value(v.String())
	} else if len(_default) > 0 {
		return config.Value(_default[0])
	}
	return ""
}

func (p RequestModal[T]) Path(ctx contracts.RequestContext, name string, _default ...string) (out config.Value) {
	if v := ctx.Param(name); v.Valid() {
		return config.Value(v.String())
	} else if len(_default) > 0 {
		return config.Value(_default[0])
	}
	return ""
}

func (p RequestModal[T]) Debug(o any) {
	fmt.Printf("%T.debug(): %+v\n", p, o)
}

func (p RequestModal[T]) Headers(ctx contracts.RequestContext, names ...string) (out map[string]string) {
	out = make(map[string]string)
	if len(names) == 0 {
		return
	}
	for _, header := range names {
		if header := strings.Trim(header, " "); header == "" {
			continue
		} else if val := ctx.Header(header); val != "" {
			out[header] = val
		}
	}
	return
}

func (p RequestModal[T]) Authorization(ctx contracts.RequestContext) (out string) {
	return ctx.Header("Authorization")
}

func (p RequestModal[T]) WithHeaderValues(ctx contracts.RequestContext, headers ...string) context.Context {
	if len(headers) == 0 {
		return ctx.Context()
	}
	headerValues := p.Headers(ctx, headers...)
	if len(headerValues) == 0 {
		return ctx.Context()
	}
	stdCtx := ctx.Context()
	for k, v := range headerValues {
		stdCtx = context.WithValue(stdCtx, k, v)
	}
	return stdCtx
}
