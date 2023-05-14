package api

import (
	"context"
	"fmt"

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
	err = r.LoadFromJsonBody(ctx, t)
	ctx.(*gin.Context).Set((*t).ContextKey(), t)
	return *t, err
}

func (r RequestModal[T]) LoadFromJsonBody(ctx context.Context, out interface{}) (err error) {
	if err = ctx.(*gin.Context).ShouldBindJSON(out); err != nil {
		return errors.Wrapf(err, "failed to unmarshall json %T from request", r)
	}
	return
}

func (r RequestModal[T]) ContextKey() string {
	var t = new(T)
	return fmt.Sprintf("%T", t)
}

func (p RequestModal[T]) ContextLoad(ctx context.Context) (out RequestParam, err error) {
	val := ctx.Value(p.ContextKey())
	if val == nil {
		return out, errors.Errorf("Failed to load %T from context with key %s", p, p.ContextKey())
	}
	return val.(RequestParam), nil
}

func (p RequestModal[T]) LoadFromContext(ctx context.Context, out RequestParam) (err error) {
	var param RequestParam
	if param, err = out.ContextLoad(ctx); err != nil {
		return errors.Wrapf(err, "Failed to load %T from context", p)
	} else if param == nil {
		param = ctx.Value(p.ContextKey()).(RequestModal[T]) //todo: finalise
	}
	utils.StructCopy(param, out)
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

func (p RequestModal[T]) Log(o any) {
	fmt.Printf("%T.log(): %+v\n", p, o)
}
