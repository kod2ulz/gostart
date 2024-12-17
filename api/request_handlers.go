package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type RoutineWithResponseFunc[T any] func(context.Context) (T, Error)

type RoutineWithListResponseFunc[T any] func(context.Context) ([]T, Error)

func BasicHandler[T any](serviceFunc RoutineWithResponseFunc[T]) gin.HandlerFunc {
	return serviceHandler(serviceFunc, func(ctx *gin.Context, out T) {
		ctx.JSON(http.StatusOK, DataResponse(out))
	})
}

func HandlerWithParam[P RequestParam](serviceFunc gin.HandlerFunc) gin.HandlerFunc {
	return genericHandlerWithParam[P](serviceFunc)
}

func ParamHandlerWithResponse[P RequestParam, T any](serviceFunc RoutineWithResponseFunc[T]) gin.HandlerFunc {
	return serviceHandlerWithParam(serviceFunc, func(ctx *gin.Context, param P, out T) {
		refs := map[string]any{}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		ctx.JSON(http.StatusOK, DataResponse(out).WithReferences(refs))
	})
}

func ParamHandlerWithFileResponse[P RequestParam](serviceFunc RoutineWithResponseFunc[FileResponse]) gin.HandlerFunc {
	return serviceHandlerWithParam(serviceFunc, fileRequestHandler[P])
}

// func HandlerWithListResponse[T any](serviceFunc RoutineWithListResponseFunc[T]) gin.HandlerFunc {
// 	return serviceHandler(serviceFunc, func(ctx *gin.Context, out []T) {
// 		ctx.JSON(http.StatusOK, ListResponse(out, Metadata{}))
// 	})
// }

func ParamHandlerWithListResponse[P RequestParam, T any](serviceFunc RoutineWithListResponseFunc[T]) gin.HandlerFunc {
	return serviceHandlerWithParam(serviceFunc, func(ctx *gin.Context, param P, res []T) {
		meta, refs := &Metadata{}, map[string]any{}
		if val, ok := ctx.Get(param.MetadataContextKey()); ok {
			meta, _ = val.(*Metadata)
		}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		ctx.JSON(http.StatusOK, ListResponse(res, *meta).WithReferences(refs))
	})
}

func serviceHandler[T any](serviceFunc func(context.Context) (T, Error), resultHandler func(*gin.Context, T)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if out, err := serviceFunc(ctx); err != nil {
			ctx.JSON(http.StatusInternalServerError, ctx.Error(err))
		} else {
			resultHandler(ctx, out)
		}
	}
}

func serviceHandlerWithParam[P RequestParam, T any](serviceFunc func(context.Context) (T, Error), successHandler func(*gin.Context, P, T)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err Error
		var param P
		if param, err = loadParamFromRequest[P](ctx); err != nil {
			ctx.JSON(err.HttpCode(), ErrorResponse[P](err))
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

func loadParamFromRequest[P RequestParam](ctx *gin.Context) (param P, err Error) {
	var e error
	var p RequestParam
	if p, e = (*new(P)).RequestLoad(ctx); e != nil {
		return param, RequestLoadError[P](errors.Wrapf(e, "failed to load %T from request", param))
	}
	ctx.Set(p.ContextKey(), p)
	if e = p.Validate(ctx); e != nil {
		return param, ValidatorError[P](errors.Wrapf(e, "validation failed for %T", param))
	}
	param = p.(P)
	return
}

func genericHandlerWithParam[P RequestParam](serviceFunc gin.HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err Error
		var param P
		if param, err = loadParamFromRequest[P](ctx); err != nil {
			ctx.JSON(err.HttpCode(), ErrorResponse[P](err))
			return
		}
		ctx.Set(param.ContextKey(), param)
		serviceFunc(ctx)
	}
}

func fileRequestHandler[P RequestParam](ctx *gin.Context, param P, out FileResponse) {
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
	ctx.Header("Content-Type", out.contentType())
	ctx.Header("Content-Disposition", "attachment; filename="+out.filename(ctx))
	if len(out.Data) > 0 {
		ctx.Header("Accept-Length", fmt.Sprint(len(out.Data)))
	}
	ctx.Writer.Write(out.Data)
	ctx.JSON(http.StatusOK, DataResponse(gin.H{
		"msg": "File downloaded successfully",
	}).WithReferences(refs))
}

/*   --  support for request unpacked from context --- */

type RequestConsumerWithResponseFunc[R RequestParam, T any] func(context.Context, R) (T, Error)

type RequestConsumerWithListResponseFunc[R RequestParam, T any] func(context.Context, R) ([]T, Error)

func requestHandlerWithParam[R RequestParam, T any](serviceFunc func(context.Context, R) (T, Error), onSuccess func(*gin.Context, R, T)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err Error
		var param R
		if param, err = loadParamFromRequest[R](ctx); err != nil {
			ctx.JSON(err.HttpCode(), ErrorResponse[R](err))
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

func RequestHandlerWithResponse[R RequestParam, T any](serviceFunc RequestConsumerWithResponseFunc[R, T]) gin.HandlerFunc {
	return requestHandlerWithParam(serviceFunc, func(ctx *gin.Context, param R, out T) {
		refs := map[string]any{}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		ctx.JSON(http.StatusOK, DataResponse(out).WithReferences(refs))
	})
}

func RequestHandlerWithFileResponse[R RequestParam](serviceFunc RequestConsumerWithResponseFunc[R, FileResponse]) gin.HandlerFunc {
	return requestHandlerWithParam(serviceFunc, fileRequestHandler[R])
}

func RequestHandlerWithListResponse[R RequestParam, T any](serviceFunc RequestConsumerWithListResponseFunc[R, T]) gin.HandlerFunc {
	return requestHandlerWithParam(serviceFunc, func(ctx *gin.Context, param R, res []T) {
		meta, refs := &Metadata{}, map[string]any{}
		if val, ok := ctx.Get(param.MetadataContextKey()); ok {
			meta, _ = val.(*Metadata)
		}
		if val, ok := ctx.Get(param.ReferencesContextKey()); ok {
			refs, _ = val.(map[string]any)
		}
		ctx.JSON(http.StatusOK, ListResponse(res, *meta).WithReferences(refs))
	})
}
