package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/kod2ulz/gostart/object"
	"github.com/kod2ulz/gostart/utils"
)

var (
// _ URLSearchParam  = (nil).(*urlSearch)
// _ URLSearchLoader = (*urlSearch).(nil)
)

type UrlFieldReader func(ctx context.Context, name string, _default ...string) (out utils.Value)

type URLSearchParam interface {
	GetFieldValues() map[string]any
	GetFieldNullables() map[string]bool
	GetFieldSort() map[string]SortType
	GetFieldComparisons() map[string]map[CompareOperator]any
	GetLimit() int64
	GetOffset() int64
	HasFieldParams() bool
	HasField(field string) bool
	HasComparison(field string, comparator CompareOperator) bool
	WithField(field string, val any) URLSearchParam
	WithTimeFormat(format string, fields ...string) URLSearchParam
	WithComparison(field string, comparator CompareOperator, val any) URLSearchParam
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
	fields      map[string]any
	sort        map[string]SortType
	null        map[string]bool
	comparisons map[string]map[CompareOperator]any
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
	if s.offset == 0 {
		if page := s.query(ctx, "page", "0").Int64(); page > 1 {
			s.offset = (page - 1) * s.limit
		}
	}
	return s
}

func (s *urlSearch) LoadFieldLookups(ctx context.Context, fields ...string) *urlSearch {
	if len(fields) == 0 {
		return s
	} else if s.fields == nil {
		s.fields = make(map[string]any)
	}
	for i := range fields {
		if val := s.query(ctx, fields[i]); val.Valid() {
			s.fields[fields[i]] = val
		} else if val = s.query(ctx, strcase.ToCamel(fields[i])); val.Valid() {
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
		s.comparisons = make(map[string]map[CompareOperator]any)
	}
	if s.null == nil {
		s.null = make(map[string]bool)
	}
	for i := range fields {
		if val := s.query(ctx, fmt.Sprintf("%s_null", fields[i])); val.Valid() {
			s.null[fields[i]] = val.Bool()
		} else if val = s.query(ctx, fmt.Sprintf("%s_null", strcase.ToCamel(fields[i]))); val.Valid() {
			s.fields[fields[i]] = val
		}
		if _, ok := s.comparisons[fields[i]]; !ok {
			s.comparisons[fields[i]] = make(map[CompareOperator]any)
		}
		for _, cp := range []CompareOperator{
			CompareGreaterThan, CompareGreaterThanOrEqual, CompareLessThan, CompareGreaterThanOrEqual, CompareNot, CompareNotEqual} {
			if val := s.query(ctx, fmt.Sprintf("%s_%s", fields[i], string(cp))); val.Valid() {
				s.comparisons[fields[i]][cp] = val
			} else if val = s.query(ctx, fmt.Sprintf("%s_%s", strcase.ToCamel(fields[i]), string(cp))); val.Valid() {
				s.fields[fields[i]] = val
			}
		}
		for _, field := range object.String(fields[i]).Variations("~%s", "~%s~", "%s~") {
			var val utils.Value
			if val = s.query(ctx, field); !val.Valid() {
				val = s.query(ctx, strcase.ToCamel(field))
			}
			if val.Valid() {
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

func (r *urlSearch) GetFieldValues() (out map[string]any) {
	if len(r.fields) == 0 {
		return map[string]any{}
	}
	return r.fields
}

func (r *urlSearch) GetFieldSort() (out map[string]SortType) {
	if len(r.sort) == 0 {
		return map[string]SortType{}
	}
	return r.sort
}

func (r *urlSearch) GetFieldComparisons() (out map[string]map[CompareOperator]any) {
	if len(r.comparisons) == 0 {
		return map[string]map[CompareOperator]any{}
	}
	return r.comparisons
}

func (r *urlSearch) GetField(name string) (out any) {
	return r.GetFieldValues()[name]
}

func (r *urlSearch) GetAnyQueryField(names ...string) (out any) {
	if len(r.fields) == 0 {
		return
	}
	for i := range names {
		if val, ok := r.fields[names[i]]; ok {
			return val
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

func (r *urlSearch) HasField(field string) bool {
	if !r.HasFieldParams() {
		return false
	}
	_, ok := r.fields[field]
	return ok
}

func (r *urlSearch) HasComparison(field string, comparator CompareOperator) (ok bool) {
	if _, ok = r.comparisons[field]; !ok {
		return false
	}
	_, ok = r.comparisons[field][comparator]
	return
}

func (r *urlSearch) WithTimeFormat(format string, fields ...string) URLSearchParam {
	if len(fields) == 0 {
		return r
	}
	for _, f := range fields {
		if _, ok := r.fields[f]; !ok {
			continue
		}
		r.replaceField(f, func(v any) any {
			if s1, ok := v.(string); ok {
				return utils.Value(s1).Time(format).UTC()
			} else if v1, ok := v.(utils.Value); ok {
				return v1.Time(format).UTC()
			}
			return v
		})
	}
	return r
}

func (r *urlSearch) replaceField(name string, modifier func(any) any) {
	if v, ok := r.fields[name]; ok {
		r.fields[name] = modifier(v)
	}
	if m, ok := r.comparisons[name]; ok {
		for c, v := range m {
			r.comparisons[name][c] = modifier(v)
		}
	}
}

func (r *urlSearch) WithField(field string, val any) URLSearchParam {
	if r.fields == nil {
		r.fields = make(map[string]any)
	}
	r.fields[field] = val
	return r
}

func (r *urlSearch) WithComparison(field string, operator CompareOperator, val any) URLSearchParam {
	if r.comparisons == nil {
		r.comparisons = make(map[string]map[CompareOperator]any)
	}
	if r.comparisons[field] == nil {
		r.comparisons[field] = make(map[CompareOperator]any)
	}
	r.comparisons[field][operator] = val
	return r
}

func WithField(param URLSearchParam, field string, val any) URLSearchParam {
	if search, ok := param.(*urlSearch); ok {
		return search.WithField(field, val)
	}
	return param
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

func WithComparison(param URLSearchParam, field string, operator CompareOperator, val any) URLSearchParam {
	search, ok := param.(*urlSearch)
	if !ok {
		return param
	}
	if search.comparisons == nil {
		search.comparisons = make(map[string]map[CompareOperator]any)
	}
	if search.comparisons[field] == nil {
		search.comparisons[field] = make(map[CompareOperator]any)
	}
	search.comparisons[field][operator] = val
	return search
}
