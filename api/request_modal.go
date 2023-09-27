package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
)

type _t struct{ RequestModal[_t] }

var _ RequestParam = RequestModal[_t]{}

type RequestModal[T RequestParam] struct{}

func (r RequestModal[T]) Validate(ctx context.Context) error {
	return utils.Validate.Struct(ctx.Value(r.ContextKey()))
}

func (r RequestModal[T]) RequestLoad(ctx context.Context) (param RequestParam, err error) {
	t := new(T)
	if err = r.LoadFromJsonBody(ctx, t); err == nil {
		ctx.(*gin.Context).Set((*t).ContextKey(), t)
		return *t, err
	}
	return nil, err
}

func (r RequestModal[T]) LoadFromJsonBody(ctx context.Context, out interface{}) (err error) {
	if err = ctx.(*gin.Context).ShouldBindJSON(out); err != nil {
		return errors.Wrapf(err, "failed to load json body into %T from request", out)
	}
	return
}

func (r RequestModal[T]) ContextKey() string {
	var t = new(T)
	return fmt.Sprintf("%T", t)
}

func (r RequestModal[T]) MetadataContextKey() string {
	return fmt.Sprintf("meta.%s", r.ContextKey())
}

func (r RequestModal[T]) ReferencesContextKey() string {
	return fmt.Sprintf("ref.%s", r.ContextKey())
}

func (r RequestModal[T]) SetResponseMetadata(ctx context.Context, meta *Metadata) (err error) {
	ctx.(*gin.Context).Set(r.MetadataContextKey(), meta)
	return
}

func (r RequestModal[T]) SetResponseReference(ctx context.Context, key string, value any) (err error) {
	var ref map[string]any
	if value == nil {
		return
	} else if val := ctx.Value(r.ReferencesContextKey()); val != nil {
		ref = val.(map[string]any)
	} else {
		ref = make(map[string]any)
	}
	ref[key] = value
	ctx.(*gin.Context).Set(r.ReferencesContextKey(), ref)
	return
}

func (p RequestModal[T]) ContextLoad(ctx context.Context) (out RequestParam, err error) {
	val := ctx.Value(p.ContextKey())
	if val == nil {
		return out, errors.Errorf("value of %T with key %s was %v in context", p, p.ContextKey(), val)
	}
	return val.(RequestParam), nil
}

func (p RequestModal[T]) LoadFromContext(ctx context.Context, out RequestParam) (err error) {
	var param RequestParam
	if out == nil {
		return errors.Errorf("out is nil")
	} else if param, err = (*new(T)).ContextLoad(ctx); err != nil {
		return errors.Wrapf(err, "Failed to load %T from context", out)
	} else if param == nil {
		if param = ctx.Value(p.ContextKey()).(RequestParam); param == nil {
			return errors.Errorf("Got %v when loading %T from context", out, out)
		}
	}
	utils.StructCopy(param, out)
	return
}

func (p RequestModal[T]) FromContext(ctx context.Context, out *T) (err error) {
	if out == nil {
		return errors.Errorf("out is nil")
	} else if val := ctx.Value(p.ContextKey()); val == nil {
		return errors.Errorf("value of %T with key %s was %v in context", *out, p.ContextKey(), val)
	} else if param, ok := val.(T); ok {
		*out = param
	} else {
		return errors.Errorf("failed to cast %T to %T ", val, *out)
	}
	return
}

func (p RequestModal[T]) InContext(ctx context.Context, in T) context.Context {
	return context.WithValue(ctx, in.ContextKey(), in)
}

func (p RequestModal[T]) Query(ctx context.Context, name string, _default ...string) (out utils.Value) {
	if v := ctx.(*gin.Context).Query(name); v != "" {
		return utils.Value(v)
	} else if len(_default) > 0 {
		return utils.Value(_default[0])
	}
	return
}

func (p RequestModal[T]) Path(ctx context.Context, name string, _default ...string) (out utils.Value) {
	if v := ctx.(*gin.Context).Param(name); v != "" {
		return utils.Value(v)
	} else if len(_default) > 0 {
		return utils.Value(_default[0])
	}
	return
}

func (p RequestModal[T]) Debug(o any) {
	fmt.Printf("%T.debug(): %+v\n", p, o)
}

func (p RequestModal[T]) Headers(ctx context.Context, names ...string) (out map[string]string) {
	out = make(map[string]string)
	if len(names) == 0 {
		return
	}
	var getHeaderValue func(string) string = func(s string) string {
		if val := ctx.Value(s); val != nil {
			return fmt.Sprint(val)
		}
		return ""
	}
	if ct, ok := ctx.(*gin.Context); ok {
		getHeaderValue = func(s string) string {
			return ct.Request.Header.Get(s)
		}
	}
	for _, header := range names {
		if header := strings.Trim(header, " "); header == "" {
			continue
		} else if val := getHeaderValue(header); val != "" {
			out[header] = fmt.Sprint(val)
		}
	}
	return
}

func (p RequestModal[T]) Authorization(ctx context.Context) (out string) {
	return ctx.(*gin.Context).Request.Header.Get("Authorization")
}
