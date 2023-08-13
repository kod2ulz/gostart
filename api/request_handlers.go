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
		ctx.JSON(http.StatusOK, DataResponse(out))
	})
}

// func HandlerWithListResponse[T any](serviceFunc RoutineWithListResponseFunc[T]) gin.HandlerFunc {
// 	return serviceHandler(serviceFunc, func(ctx *gin.Context, out []T) {
// 		ctx.JSON(http.StatusOK, ListResponse(out, Metadata{}))
// 	})
// }

func ParamHandlerWithListResponse[P RequestParam, T any](serviceFunc RoutineWithListResponseFunc[T]) gin.HandlerFunc {
	fmt.Printf("inside param handler\n")
	return serviceHandlerWithParam(serviceFunc, func(ctx *gin.Context, param P, res []T) {
		fmt.Printf("inside service handler finc callback\n")
		if val, ok := ctx.Get(param.MetadataContextKey()); ok {
			if meta, ok := val.(*Metadata); ok {
				ctx.JSON(http.StatusOK, ListResponse(res, *meta))
				return
			}
		}
		ctx.JSON(http.StatusOK, ListResponse(res, Metadata{}))
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
	fmt.Printf("inside service handler with param handler\n")
	return func(ctx *gin.Context) {
		var err Error
		var param P
		fmt.Printf("loading param from request <<-\n")
		if param, err = loadParamFromRequest[P](ctx); err != nil {
			ctx.JSON(err.http(), ErrorResponse[P](err))
			return
		}
		ctx.Set(param.ContextKey(), param)
		if out, err := serviceFunc(ctx); err != nil {
			ctx.JSON(err.http(), err)
		} else {
			successHandler(ctx, param, out)
		}
	}
}

func loadParamFromRequest[P RequestParam](ctx *gin.Context) (param P, err Error) {
	var e error
	var p RequestParam
	fmt.Printf("loading from request \n")
	if p, e = (*new(P)).RequestLoad(ctx); e != nil {
		return param, RequestLoadError[P](errors.Wrapf(e, "failed to load %T from request", param))
	}
	fmt.Printf("writing loaded data to context\n")
	ctx.Set(p.ContextKey(), p)
	fmt.Printf("running %T.Validate() \n", )
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
			ctx.JSON(err.http(), ErrorResponse[P](err))
			return
		}
		ctx.Set(param.ContextKey(), param)
		serviceFunc(ctx)
	}
}
