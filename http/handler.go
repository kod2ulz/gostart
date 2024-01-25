package http

import (
	"context"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
)

type Param interface {
	api.RequestParam
	ContextKey() string
}

func GenericHandler[P Param, R any](ctx context.Context, log *logr.Logger, method, url string) func(context.Context) (out R, err api.Error) {
	var err error
	var res api.Response[R]
	var param api.RequestParam

	return func(ctx context.Context) (out R, apiErr api.Error) {
		if param, err = (*new(P)).ContextLoad(ctx); err != nil {
			return out, api.RequestLoadError[P](errors.Wrapf(err, "Failed to load %T from context", param))
		} else if param == nil {
			var ok bool
			if param, ok = ctx.Value((*new(P)).ContextKey()).(P); !ok {
				return out, api.RequestLoadError[P](errors.Errorf("Got %v when loading %T from context", param, param))
			}
		}
		if res = Client[R](log.Entry).Body(&param).Request(ctx, method, url); res.HasError() {
			return out, res.Error
		} else if err = res.ParseDataTo(&out); err != nil {
			return out, api.GeneralError[R](errors.Wrapf(err, "failed to parse %T to %T", res.Data, out)).
				WithError(errors.New("encountered error generating verification response")).
				WithErrorCode(api.ErrorCodeServiceError)
		}
		return
	}
}
