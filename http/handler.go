package http

import (
	"context"
	"strings"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
	"github.com/sirupsen/logrus"
)

type Param interface {
	contracts.RequestParam
	ContextKey() string
}

func GenericHandler[P Param, R any](log *logrus.Entry, method string, url ...string) func(context.Context) (out R, err ierrors.Error) {
	var err error
	var res contracts.Response[R]
	var param contracts.RequestParam

	return func(ctx context.Context) (out R, apiErr ierrors.Error) {
		if len(url) == 0 {
			return out, errors.RequestLoadFailed[P](errors.Errorf("invalid url"))
		} else if param, err = (*new(P)).ContextLoad(ctx); err != nil {
			return out, errors.RequestLoadFailed[P](errors.Wrapf(err, "Failed to load %T from context", param))
		} else if param == nil {
			var ok bool
			if param, ok = ctx.Value((*new(P)).ContextKey()).(P); !ok {
				return out, errors.RequestLoadFailed[P](errors.Errorf("Got %v when loading %T from context", param, param))
			}
		}
		if res = Client[R](log).Body(&param).Request(ctx, method, strings.Join(url, "/")); res.HasError() {
			return out, res.Error
		} else if err = res.ParseDataTo(&out); err != nil {
			return out, errors.GeneralFailure[R](errors.Wrapf(err, "failed to parse %T to %T", res.Data, out)).
				WithError(errors.Errorf("encountered error generating verification response")).
				WithErrorCode(errors.ErrorCodeServiceError)
		}
		return
	}
}
