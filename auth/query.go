package auth

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
)

type ListQueryRequest struct {
	api.RequestModal[ListQueryRequest]
	User   User      `json:"-"`
	ID     uuid.UUID `json:"-"`
	Limit  int       `json:"limit" validate:"required"`
	Offset int       `json:"offset"`
	Query  string    `json:"query"`
}

func (r ListQueryRequest) RequestLoad(ctx context.Context) (param api.RequestParam, err error) {
	var user User
	var id uuid.UUID
	if user, err = GetUser(ctx); err != nil {
		return
	} else if pid := ctx.(*gin.Context).Param("id"); pid != "" {
		id, _ = uuid.Parse(pid)
	}
	return ListQueryRequest{
		ID:     id,
		Limit:  api.QueryFromContext(ctx, "limit", "20").Int(),
		Offset: api.QueryFromContext(ctx, "offset", "0").Int(),
		Query:  ctx.(*gin.Context).Query("query"),
		User:   user,
	}, err
}

type UuidParamRequest struct {
	api.RequestModal[UuidParamRequest]
	User User      `json:"-"`
	ID   uuid.UUID `json:"id,omitempty" validate:"required"`
}

func (r UuidParamRequest) RequestLoad(ctx context.Context) (param api.RequestParam, err error) {
	var user User
	var id uuid.UUID
	if user, err = GetUser(ctx); err != nil {
		return
	} else if id, err = uuid.Parse(ctx.(*gin.Context).Param("id")); err != nil {
		return param, errors.Wrap(err, "failed to parse ID from path")
	}
	return UuidParamRequest{ID: id, User: user}, err
}

type UserListParamRequest struct {
	ID          uuid.UUID      `json:"id" validate:"required"`
	AddUsers    utils.UuidList `json:"addUsers"`
	RemoveUsers utils.UuidList `json:"removeUsers"`
	Users       utils.UuidList `json:"users"`

	User User `json:"-"`
	api.RequestModal[UserListParamRequest]
}

func (r UserListParamRequest) Validate(ctx context.Context) error {
	if len(r.AddUsers) == 0 && len(r.RemoveUsers) == 0 {
		return errors.New("both addUsers and removeUsers cannot be blank")
	}
	return utils.Validate.Struct(r)
}

func (r UserListParamRequest) RequestLoad(ctx context.Context) (param api.RequestParam, err error) {
	var user User
	var id uuid.UUID
	var out UserListParamRequest
	if user, err = GetUser(ctx); err != nil {
		return
	} else if id, err = uuid.Parse(ctx.(*gin.Context).Param("id")); err != nil {
		return param, errors.Wrap(err, "failed to parse ID from path")
	} else if err = ctx.(*gin.Context).ShouldBindJSON(&out); err != nil {
		return param, errors.Wrapf(err, "failed to initialise %T from request", out)
	}
	out.User, out.ID = user, id
	return out, err
}