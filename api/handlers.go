package api

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
)

var (
	ErrorNotFound = errors.New("Not Found")
)

type GetServiceFunc[T any] func(context.Context) (T, error)

type GetListServiceFunc[T any] func(context.Context) ([]T, error)

type GetListResponseServiceFunc[T any] func(context.Context) (ListResponse[T], error)

func GetHandler[T any](serviceFunc GetServiceFunc[T]) gin.HandlerFunc {
	return serviceHandler(serviceFunc)
}

func HandleWithParam[P RequestParam](serviceFunc gin.HandlerFunc) gin.HandlerFunc {
	return handlerWithParam[P](serviceFunc)
}

func GetWithParamHandler[P RequestParam, T any](serviceFunc GetServiceFunc[T]) gin.HandlerFunc {
	return serviceWithParamHandler[P](serviceFunc)
}

func GetListHandler[T any](serviceFunc GetListServiceFunc[T]) gin.HandlerFunc {
	return serviceHandler(serviceFunc)
}

func GetWithParamListHandler[P RequestParam, T any](serviceFunc GetListServiceFunc[T]) gin.HandlerFunc {
	return serviceWithParamHandler[P](serviceFunc)
}

func serviceHandler[T any](serviceFunc func(context.Context) (T, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if out, err := serviceFunc(ctx); err != nil {
			ctx.JSON(http.StatusInternalServerError, ctx.Error(err))
		} else {
			ctx.JSON(http.StatusOK, out)
		}
	}
}

func serviceWithParamHandler[P RequestParam, T any](serviceFunc func(context.Context) (T, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		var param P
		if param, err = _getParamFromRequest[P](ctx); err != nil {
			return
		}
		ctx.Set(param.ContextKey(), param)
		if out, err := serviceFunc(ctx); err != nil {
			if errors.Is(err, ErrorNotFound) || utils.Error.SqlNoRows(err) {
				ctx.JSON(http.StatusNotFound, ctx.Error(errors.Wrapf(err, "%T", out)))
			} else if errors.Is(err, sql.ErrNoRows) {
				ctx.JSON(http.StatusOK, []T{})
			} else {
				ctx.JSON(http.StatusInternalServerError, ctx.Error(err))
			}
		} else {
			ctx.JSON(http.StatusOK, out)
		}
	}
}

func _getParamFromRequest[P RequestParam](ctx *gin.Context) (param P, err error) {
	var p RequestParam
	if p, err = (*new(P)).RequestLoad(ctx); err != nil {
		ctx.JSON(http.StatusBadRequest, ctx.Error(errors.Wrapf(err, "failed to load %T from request", param)))
		return
	}
	ctx.Set(p.ContextKey(), p)
	if err = p.Validate(ctx); err != nil {
		if isBodyValidationError(ctx, err) {
			ctx.JSON(http.StatusBadRequest, formatErrors(
				ctx.Error(errors.Errorf("validation failed for %T", param)),
				[]*gin.Error(ctx.Errors)...))
		} else {
			ctx.JSON(http.StatusBadRequest, ctx.Error(errors.Wrapf(err, "validation failed for %T", param)))
		}
		return
	} else {
		param = p.(P)
	}
	return
}

func handlerWithParam[P RequestParam](serviceFunc gin.HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		var param P
		if param, err = _getParamFromRequest[P](ctx); err != nil {
			return
		}
		ctx.Set(param.ContextKey(), param)
		serviceFunc(ctx)
	}
}

func isBodyValidationError(ctx *gin.Context, err error) bool {
	if err == nil || !strings.Contains(err.Error(), "Key") {
		return false
	}
	for _, msg := range strings.Split(err.Error(), "Key:") {
		ctx.Error(errors.New(strings.Trim(msg, "\n ")))
	}
	return true
}

func formatErrors(err *gin.Error, errs ...*gin.Error) (out map[string]interface{}) {
	if err == nil || len(errs) == 0 {
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		} else {
			return map[string]interface{}{}
		}
	}
	out = map[string]interface{}{"error": err.Error()}
	errMsgs := make(map[string]string, 0)
	for i := range errs {
		if errs[i] != nil && !errors.Is(err, errs[i]) && errs[i].Error() != "" {
			if !strings.Contains(errs[i].Error(), " Error:") {
				errMsgs[""] = errs[i].Error()
			} else {
				parts := strings.Split(errs[i].Error(), " Error:")
				errMsgs[strings.Trim(parts[0], " '")] = strings.Trim(parts[1], " ")
			}
		}
	}
	if len(errMsgs) > 0 {
		out["meta"] = errMsgs
	}
	return
}
