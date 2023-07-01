package query

type SortType string

const (
	SortAsc  SortType = "asc"
	SortDesc SortType = "desc"
)

func sortTypeValid(str string) bool {
	for _, st := range []SortType{SortAsc, SortDesc} {
		if str == string(st) {
			return true
		}
	}
	return false
}

type SortConsumer interface {
	addFieldSort(sort SortType, fields ...string)
}

type SortFunc func(SortConsumer)

func Asc(fields ...string) SortFunc {
	return func(sc SortConsumer) { sc.addFieldSort(SortAsc, fields...) }
}

func Desc(fields ...string) SortFunc {
	return func(sc SortConsumer) { sc.addFieldSort(SortDesc, fields...) }
}

func UrlFieldSort(p URLSearchParam) SortFunc {
	return func(sc SortConsumer) {
		if len(p.GetFieldSort()) == 0 {
			return
		}
		for field, sort := range p.GetFieldSort() {
			sc.addFieldSort(sort, field)
		}
	}
}
