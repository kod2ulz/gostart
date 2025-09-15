package query

type SortType string

const (
	SortAsc   SortType = "asc"
	SortDesc  SortType = "desc"
	SortReset SortType = "clear"
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

func NoSort() SortFunc {
	return func(sc SortConsumer) { sc.addFieldSort(SortReset) }
}

func UrlFieldSort(p URLSearchParam) SortFunc {
	return func(sc SortConsumer) {
		if len(p.GetSorts()) == 0 {
			return
		}
		for _, s := range p.GetSorts() {
			sc.addFieldSort(s.Type, s.DBName)
		}
	}
}
