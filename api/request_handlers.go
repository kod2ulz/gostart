package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

// ginContextAdapter provides a minimal RequestContext implementation for gin.Context
type ginContextAdapter struct {
	ctx *gin.Context
}

func (g *ginContextAdapter) Query(key string, defaultValue ...string) contracts.Value {
	if val := g.ctx.Query(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

func (g *ginContextAdapter) Param(key string, defaultValue ...string) contracts.Value {
	if val := g.ctx.Param(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

func (g *ginContextAdapter) Header(key string) string {
	return g.ctx.Request.Header.Get(key)
}

func (g *ginContextAdapter) ShouldBindJSON(obj interface{}) error {
	return g.ctx.ShouldBindJSON(obj)
}

func (g *ginContextAdapter) Context() context.Context {
	return g.ctx
}

func (g *ginContextAdapter) Value(key interface{}) interface{} {
	return g.ctx.Value(key)
}

func (g *ginContextAdapter) Set(key string, value interface{}) {
	g.ctx.Set(key, value)
}

type RoutineWithResponseFunc[T any] func(context.Context) (T, ierrors.Error)

type RoutineWithListResponseFunc[T any] func(context.Context) ([]T, ierrors.Error)

func BasicHandler[T any](serviceFunc RoutineWithResponseFunc[T]) gin.HandlerFunc {
	return serviceHandler(serviceFunc, func(ctx *gin.Context, out T) {
		ctx.JSON(http.StatusOK, contracts.DataResponse(out))
	})
}

func HandlerWithParam[P contracts.RequestParam](serviceFunc gin.HandlerFunc) gin.HandlerFunc {
	return genericHandlerWithParam[P](serviceFunc)
}

func ParamHandlerWithResponse[P contracts.RequestParam, T any](serviceFunc RoutineWithResponseFunc[T]) gin.HandlerFunc {
	return serviceHandlerWithParam(serviceFunc, func(ctx *gin.Context, param P, out T) {
		refs := map[string]any{}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		ctx.JSON(http.StatusOK, contracts.DataResponse(out).WithReferences(refs))
	})
}

func ParamHandlerWithResponseFinalser[P contracts.RequestParam, T any](serviceFunc RoutineWithResponseFunc[T], finaliser func (*gin.Context, T) gin.HandlerFunc) gin.HandlerFunc {
	return serviceHandlerWithParam(serviceFunc, func(ctx *gin.Context, param P, out T) {
		refs := map[string]any{}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		} 
		if finaliser != nil {
			finaliser(ctx, out)
		} else {
			ctx.JSON(http.StatusOK, contracts.DataResponse(out).WithReferences(refs))
		}
	})
}

func ParamHandlerWithFileResponse[P contracts.RequestParam](serviceFunc RoutineWithResponseFunc[contracts.FileResponse]) gin.HandlerFunc {
	return serviceHandlerWithParam(serviceFunc, fileRequestHandler[P])
}

// func HandlerWithListResponse[T any](serviceFunc RoutineWithListResponseFunc[T]) gin.HandlerFunc {
// 	return serviceHandler(serviceFunc, func(ctx *gin.Context, out []T) {
// 		ctx.JSON(http.StatusOK, ListResponse(out, Metadata{}))
// 	})
// }

func ParamHandlerWithListResponse[P contracts.RequestParam, T any](serviceFunc RoutineWithListResponseFunc[T]) gin.HandlerFunc {
	return serviceHandlerWithParam(serviceFunc, func(ctx *gin.Context, param P, res []T) {
		meta, refs := &contracts.Metadata{}, map[string]any{}
		if val, ok := ctx.Get(param.MetadataContextKey()); ok {
			meta, _ = val.(*contracts.Metadata)
		}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		ctx.JSON(http.StatusOK, contracts.ListResponse(res, *meta).WithReferences(refs))
	})
}

func serviceHandler[T any](serviceFunc func(context.Context) (T, ierrors.Error), resultHandler func(*gin.Context, T)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if out, err := serviceFunc(ctx); err != nil {
			ctx.JSON(http.StatusInternalServerError, ctx.Error(err))
		} else {
			resultHandler(ctx, out)
		}
	}
}

func serviceHandlerWithParam[P contracts.RequestParam, T any](serviceFunc func(context.Context) (T, ierrors.Error), successHandler func(*gin.Context, P, T)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err ierrors.Error
		var param P
		if param, err = loadParamFromRequest[P](ctx); err != nil {
			ctx.JSON(err.HttpCode(), contracts.ErrorResponse[P](err))
			return
		}
		ctx.Set(param.ContextKey(), param)
		if out, err := serviceFunc(ctx); err != nil {
			ctx.JSON(err.HttpCode(), err)
		} else {
			successHandler(ctx, param, out)
		}
	}
}

func loadParamFromRequest[P contracts.RequestParam](ctx *gin.Context) (param P, err ierrors.Error) {
	var e error
	var p contracts.RequestParam
	adapterCtx := &ginContextAdapter{ctx}
	if p, e = (*new(P)).RequestLoad(adapterCtx); e != nil {
		return param, errors.RequestLoadError[P](errors.Wrapf(e, "failed to load %T from request", param))
	}
	ctx.Set(p.ContextKey(), p)
	if e = p.Validate(adapterCtx); e != nil {
		return param, errors.ValidatorError[P](errors.Wrapf(e, "validation failed for %T", param))
	}
	param = p.(P)
	return
}

func genericHandlerWithParam[P contracts.RequestParam](serviceFunc gin.HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err ierrors.Error
		var param P
		if param, err = loadParamFromRequest[P](ctx); err != nil {
			ctx.JSON(err.HttpCode(), contracts.ErrorResponse[P](err))
			return
		}
		ctx.Set(param.ContextKey(), param)
		serviceFunc(ctx)
	}
}

func fileRequestHandler[P contracts.RequestParam](ctx *gin.Context, param P, out contracts.FileResponse) {
	refs := map[string]any{}
	if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
		if refs, ok = val.(map[string]any); ok {
			for k, v := range refs {
				if hval, ok := v.(string); ok {
					ctx.Header(k, hval)
				}
			}
		}
	}
	ctx.Header("Content-Type", out.GetContentType())
	ctx.Header("Content-Disposition", "attachment; filename="+out.GetFilename(&ginContextAdapter{ctx}))
	if len(out.Data) > 0 {
		ctx.Header("Accept-Length", fmt.Sprint(len(out.Data)))
	}
	ctx.Writer.Write(out.Data)
	// ctx.JSON(http.StatusOK, DataResponse(gin.H{
	// 	"msg": "File downloaded successfully",
	// }).WithReferences(refs))
}

/*   --  support for request unpacked from context --- */

type RequestConsumerWithResponseFunc[R contracts.RequestParam, T any] func(context.Context, R) (T, ierrors.Error)

type RequestConsumerWithListResponseFunc[R contracts.RequestParam, T any] func(context.Context, R) ([]T, ierrors.Error)

func requestHandlerWithParam[R contracts.RequestParam, T any](serviceFunc func(context.Context, R) (T, ierrors.Error), onSuccess func(*gin.Context, R, T)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err ierrors.Error
		var param R
		if param, err = loadParamFromRequest[R](ctx); err != nil {
			ctx.JSON(err.HttpCode(), contracts.ErrorResponse[R](err))
			return
		}
		ctx.Set(param.ContextKey(), param)
		if out, err := serviceFunc(ctx, param); err != nil {
			ctx.JSON(err.HttpCode(), err)
		} else {
			onSuccess(ctx, param, out)
		}
	}
}

func RequestHandlerWithResponse[R contracts.RequestParam, T any](serviceFunc RequestConsumerWithResponseFunc[R, T]) gin.HandlerFunc {
	return requestHandlerWithParam(serviceFunc, func(ctx *gin.Context, param R, out T) {
		refs := map[string]any{}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		ctx.JSON(http.StatusOK, contracts.DataResponse(out).WithReferences(refs))
	})
}

func RequestHandlerWithResponseFinalizer[R contracts.RequestParam, T any](serviceFunc RequestConsumerWithResponseFunc[R, T], finaliser func (*gin.Context, T)) gin.HandlerFunc {
	return requestHandlerWithParam(serviceFunc, func(ctx *gin.Context, param R, out T) {
		refs := map[string]any{}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		if finaliser != nil {
			finaliser(ctx, out)
		} else {
			ctx.JSON(http.StatusOK, contracts.DataResponse(out).WithReferences(refs))
		}
	})
}

func RequestHandlerWithFileResponse[R contracts.RequestParam](serviceFunc RequestConsumerWithResponseFunc[R, contracts.FileResponse]) gin.HandlerFunc {
	return requestHandlerWithParam(serviceFunc, fileRequestHandler[R])
}

func RequestHandlerWithListResponse[R contracts.RequestParam, T any](serviceFunc RequestConsumerWithListResponseFunc[R, T]) gin.HandlerFunc {
	return requestHandlerWithParam(serviceFunc, func(ctx *gin.Context, param R, res []T) {
		meta, refs := &contracts.Metadata{}, map[string]any{}
		if val, ok := ctx.Get(param.MetadataContextKey()); ok {
			meta, _ = val.(*contracts.Metadata)
		}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		ctx.JSON(http.StatusOK, contracts.ListResponse(res, *meta).WithReferences(refs))
	})
}
