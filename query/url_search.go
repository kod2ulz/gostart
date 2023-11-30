package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/kod2ulz/gostart/object"
	"github.com/kod2ulz/gostart/utils"
)

type UrlFieldReader func(ctx context.Context, name string, _default ...string) (out utils.Value)

type URLSearchParam interface {
	GetFieldValues() map[string]utils.Value
	GetFieldNullables() map[string]bool
	GetFieldSort() map[string]SortType
	GetFieldComparisons() map[string]map[CompareOperator]utils.Value
	GetLimit() int64
	GetOffset() int64
	HasFieldParams() bool
}

type URLSearchLoader interface {
	Load(ctx context.Context, fields ...string) URLSearchLoader
	LoadBoundaries(ctx context.Context) URLSearchLoader
	LoadFieldSort(ctx context.Context, fields ...string) URLSearchLoader
	LoadFieldLookups(ctx context.Context, fields ...string) URLSearchLoader
	LoadFieldComparisons(ctx context.Context, fields ...string) URLSearchLoader
}

func SearchUrl(queryReader UrlFieldReader) *urlSearch {
	return &urlSearch{query: queryReader}
}

type urlSearch struct {
	limit       int64
	offset      int64
	fields      map[string]utils.Value
	sort        map[string]SortType
	null        map[string]bool
	comparisons map[string]map[CompareOperator]utils.Value
	query       UrlFieldReader
}

func (s *urlSearch) Load(ctx context.Context, fields ...string) *urlSearch {
	return s.LoadBoundaries(ctx).
		LoadFieldSort(ctx, fields...).
		LoadFieldLookups(ctx, fields...).
		LoadFieldComparisons(ctx, fields...)
}

func (s *urlSearch) LoadFieldSort(ctx context.Context, fields ...string) *urlSearch {
	if len(fields) == 0 {
		return s
	} else if s.sort == nil {
		s.sort = make(map[string]SortType)
	}
	for i := range fields {
		if val := s.query(ctx, "sort_"+fields[i]); val.Valid() && sortTypeValid(val.String()) {
			s.sort[fields[i]] = SortType(val.String())
		}
	}
	return s
}

func (s *urlSearch) LoadBoundaries(ctx context.Context) *urlSearch {
	s.limit = s.query(ctx, "limit", fmt.Sprint(SELECT_LIMIT)).Int64()
	s.offset = s.query(ctx, "offset", "0").Int64()
	return s
}

func (s *urlSearch) LoadFieldLookups(ctx context.Context, fields ...string) *urlSearch {
	if len(fields) == 0 {
		return s
	} else if s.fields == nil {
		s.fields = make(map[string]utils.Value)
	}
	for i := range fields {
		if val := s.query(ctx, fields[i]); val.Valid() {
			s.fields[fields[i]] = val
		}
	}
	return s
}

func (s *urlSearch) LoadFieldComparisons(ctx context.Context, fields ...string) *urlSearch {
	if len(fields) == 0 {
		return s
	}
	if s.comparisons == nil {
		s.comparisons = make(map[string]map[CompareOperator]utils.Value)
	}
	if s.null == nil {
		s.null = make(map[string]bool)
	}
	for i := range fields {
		if val := s.query(ctx, fmt.Sprintf("%s_null", fields[i])); val.Valid() {
			s.null[fields[i]] = val.Bool()
		}
		if _, ok := s.comparisons[fields[i]]; !ok {
			s.comparisons[fields[i]] = make(map[CompareOperator]utils.Value)
		}
		for _, cp := range []CompareOperator{
			CompareGreaterThan, CompareGreaterThanOrEqual, CompareLessThan, CompareGreaterThanOrEqual, CompareNot, CompareNotEqual} {
			if val := s.query(ctx, fmt.Sprintf("%s_%s", fields[i], string(cp))); val.Valid() {
				s.comparisons[fields[i]][cp] = val
			}
		}
		for _, field := range object.String(fields[i]).Variations("~%s", "~%s~", "%s~") {
			if val := s.query(ctx, field); val.Valid() {
				s.comparisons[fields[i]][CompareLike] = utils.Value(strings.Replace(strings.ReplaceAll(field, "~", "%"), fields[i], val.String(), 1))
				break
			}
		}
	}
	return s
}

func (r *urlSearch) GetFieldNullables() (out map[string]bool) {
	if len(r.fields) == 0 {
		return map[string]bool{}
	}
	return r.null
}

func (r *urlSearch) GetFieldValues() (out map[string]utils.Value) {
	if len(r.fields) == 0 {
		return map[string]utils.Value{}
	}
	return r.fields
}

func (r *urlSearch) GetFieldSort() (out map[string]SortType) {
	if len(r.sort) == 0 {
		return map[string]SortType{}
	}
	return r.sort
}

func (r *urlSearch) GetFieldComparisons() (out map[string]map[CompareOperator]utils.Value) {
	if len(r.comparisons) == 0 {
		return map[string]map[CompareOperator]utils.Value{}
	}
	return r.comparisons
}

func (r *urlSearch) GetField(name string) (out utils.Value) {
	return r.GetFieldValues()[name]
}

func (r *urlSearch) GetAnyQueryField(names ...string) (out utils.Value) {
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

func (r *urlSearch) GetLimit() int64 {
	return r.limit
}

func (r *urlSearch) GetOffset() int64 {
	return r.offset
}

func (r *urlSearch) HasFieldParams() bool {
	return len(r.fields)+len(r.sort)+len(r.comparisons) > 0
}

func WithField(param URLSearchParam, field string, val utils.Value) URLSearchParam {
	search, ok := param.(*urlSearch)
	if !ok {
		return param
	} else if search.fields == nil {
		search.fields = make(map[string]utils.Value)
	}
	search.fields[field] = val
	return search
}

func WithSort(param URLSearchParam, field string, sort SortType) URLSearchParam {
	search, ok := param.(*urlSearch)
	if !ok {
		return param
	} else if search.sort == nil {
		search.sort = make(map[string]SortType)
	}
	search.sort[field] = sort
	return search
}

func WithComparison(param URLSearchParam, field string, operator CompareOperator, val utils.Value) URLSearchParam {
	search, ok := param.(*urlSearch)
	if !ok {
		return param
	}
	if search.comparisons == nil {
		search.comparisons = make(map[string]map[CompareOperator]utils.Value)
	}
	if search.comparisons[field] == nil {
		search.comparisons[field] = make(map[CompareOperator]utils.Value)
	}
	search.comparisons[field][operator] = val
	return search
}
