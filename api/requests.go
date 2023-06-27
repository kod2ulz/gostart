package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

type FieldSortType string
type FieldCompareType string

const (
	SortAsc  FieldSortType = "asc"
	SortDesc FieldSortType = "desc"
)
const (
	CompareGreaterThan        FieldCompareType = "gt"
	CompareGreaterThanOrEqual FieldCompareType = "gte"
	CompareLessThan           FieldCompareType = "lt"
	CompareLessThanOrEqual    FieldCompareType = "lte"
	CompareNot                FieldCompareType = "not"
)

type FieldQueryParamProvider interface {
	GetQueryFieldValues() map[string]utils.Value
	GetQueryFieldSort() map[string]FieldSortType
	GetQueryFieldComparisons() map[string]map[FieldCompareType]utils.Value
	GetLimit() int32
	GetOffset() int32
	HasQueryFieldParams() bool
}

type ListRequest struct {
	User   User  `json:"-" validate:"required"`
	Limit  int32 `validate:"required,gte=1"`
	Offset int32 `validate:"omitempty,gte=0"`
	RequestModal[ListRequest]

	fields      map[string]utils.Value
	sort        map[string]FieldSortType
	comparisons map[string]map[FieldCompareType]utils.Value
}

func (r ListRequest) Metadata() *Metadata {
	return &Metadata{
		Current: int64(r.Offset), Limit: int64(r.Limit), Offset: int64(r.Offset),
	}
}

func (r ListRequest) DefaultMetadata(ctx context.Context) (out *Metadata) {
	out = r.Metadata()
	r.SetResponseMetadata(ctx, out)
	return
}

func (r ListRequest) RequestLoad(ctx context.Context) (param RequestParam, err error) {
	var out ListRequest = ListRequest{}
	if out.User, err = GetUser(ctx); err != nil {
		return
	}
	out.Limit = int32(out.Query(ctx, "limit", "20").Int())
	out.Offset = int32(out.Query(ctx, "offset", "0").Int())
	ctx.(*gin.Context).Set(out.ContextKey(), &out)
	return out, err
}

func (r *ListRequest) LoadQueryFields(ctx context.Context, names ...string) *ListRequest {
	if len(names) == 0 {
		return r
	} else if r.fields == nil {
		r.fields = make(map[string]utils.Value)
	}
	for i := range names {
		if val := r.Query(ctx, names[i]); val.Valid() {
			r.fields[names[i]] = val
		}
	}
	return r
}

func (r *ListRequest) LoadQuerySort(ctx context.Context, names ...string) *ListRequest {
	if len(names) == 0 {
		return r
	} else if r.sort == nil {
		r.sort = make(map[string]FieldSortType)
	}
	for i := range names {
		if val := r.Query(ctx, "sort_"+names[i]); val.Valid() {
			r.sort[names[i]] = FieldSortType(val.String())
		}
	}
	return r
}

func (r *ListRequest) LoadQueryComparisons(ctx context.Context, names ...string) *ListRequest {
	if len(names) == 0 {
		return r
	} else if r.comparisons == nil {
		r.comparisons = make(map[string]map[FieldCompareType]utils.Value)
	}
	for i := range names {
		for _, cp := range []FieldCompareType{
			CompareGreaterThan, CompareGreaterThanOrEqual, CompareLessThan, CompareGreaterThanOrEqual, CompareNot} {
			if val := r.Query(ctx, fmt.Sprintf("%s_%s", names[i], string(cp))); val.Valid() {
				if _, ok := r.comparisons[names[i]]; !ok {
					r.comparisons[names[i]] = make(map[FieldCompareType]utils.Value)
				}
				r.comparisons[names[i]][cp] = val
			}
		}
	}
	return r
}

func (r ListRequest) GetQueryFieldValues() (out map[string]utils.Value) {
	if len(r.fields) == 0 {
		return map[string]utils.Value{}
	}
	return r.fields
}

func (r ListRequest) GetQueryFieldSort() (out map[string]FieldSortType) {
	if len(r.sort) == 0 {
		return map[string]FieldSortType{}
	}
	return r.sort
}

func (r ListRequest) GetQueryFieldComparisons() (out map[string]map[FieldCompareType]utils.Value) {
	if len(r.comparisons) == 0 {
		return map[string]map[FieldCompareType]utils.Value{}
	}
	return r.comparisons
}

func (r ListRequest) GetQueryField(name string) (out utils.Value) {
	return r.GetQueryFieldValues()[name]
}

func (r ListRequest) GetAnyQueryField(names ...string) (out utils.Value) {
	if len(r.fields) == 0 {
		return
	}
	for i := range names {
		if val, ok := r.fields[names[i]]; ok {
			return utils.Value(val)
		}
	}
	return
}

func (r ListRequest) GetLimit() int32 {
	return r.Limit
}

func (r ListRequest) GetOffset() int32 {
	return r.Offset
}

func (r ListRequest) HasQueryFieldParams() bool {
	return len(r.fields)+len(r.sort)+len(r.comparisons) > 0
}

type ListRequestIdType interface {
	string | uuid.UUID | constraints.Integer
}

type ListRequestWithID[ID ListRequestIdType] struct {
	ID ID `validate:"required"`
	ListRequest
	RequestModal[ListRequestWithID[ID]]
}

func (r *ListRequestWithID[ID]) setId(id interface{}) {
	r.ID = id.(ID)
}

func (r ListRequestWithID[ID]) RequestLoad(ctx context.Context) (param RequestParam, err error) {
	var _id ID
	var pathId string
	var out ListRequestWithID[ID] = ListRequestWithID[ID]{ListRequest: ListRequest{}}
	if p, e := out.ListRequest.RequestLoad(ctx); e != nil {
		return param, RequestLoadError[ListRequestWithID[ID]](errors.Wrapf(e, "failed to load %T from request", r))
	} else if pathId = ctx.(*gin.Context).Param("id"); pathId == "" {
		return param, errors.Errorf("could not load path parameter value with key:id")
	} else {
		out.ListRequest = p.(ListRequest)
	}
	switch any(_id).(type) {
	case string:
		out.setId(pathId)
	case uuid.UUID:
		var uid uuid.UUID
		if uid, err = uuid.Parse(pathId); err != nil {
			return param, errors.Wrapf(err, "could not parse path parameter value with key:id to %T", _id)
		}
		out.setId(uid)
	case int64, uint64:
		i, _ := strconv.ParseInt(pathId, 10, 64)
		out.setId(i)
	default:
		i, _ := strconv.Atoi(pathId)
		out.setId(i)
	}
	ctx.(*gin.Context).Set(out.ContextKey(), out)
	return out, err
}
