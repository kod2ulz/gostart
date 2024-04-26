package query

import (
	"context"

	"github.com/kod2ulz/gostart/collections"
)

type UrlFields struct {
	// Comparable are fields that are used as filter criteria compared with a like query
	// e.g name_null=true, age_gt=24, name~=joan > joan,joanita,...
	// fields in this case are [name, age]
	Comparable collections.List[string]

	// Lookup are fields that try to find an exact match,
	// e.g name=john
	// fields in this case are [name]
	Lookup collections.List[string]

	// Sort are fields that can be used to order the result
	// e.g. sort_name=asc
	// fields in this case are [name]
	Sort collections.List[string]
}

func (u *UrlFields) SearchParams(ctx context.Context, queryReader UrlFieldReader) (out URLSearchParam) {
	search := SearchUrl(queryReader)
	if len(u.Comparable) > 0 {
		search.LoadFieldComparisons(ctx, u.Comparable...)
	}
	if len(u.Lookup) > 0 {
		search.LoadFieldLookups(ctx, u.Lookup...)
	}
	if len(u.Sort) > 0 {
		search.LoadFieldSort(ctx, u.Sort...)
	}
	return search.LoadBoundaries(ctx)
}
