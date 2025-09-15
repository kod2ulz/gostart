package query

import (
	"fmt"
	"strings"

	"github.com/kod2ulz/gostart/utils"
)

type CompareOperator string

const (
	CompareEqual              CompareOperator = "eq"
	CompareGreaterThan        CompareOperator = "gt"
	CompareGreaterThanOrEqual CompareOperator = "gte"
	CompareLessThan           CompareOperator = "lt"
	CompareLessThanOrEqual    CompareOperator = "lte"
	CompareNot                CompareOperator = "not"
	CompareNotEqual           CompareOperator = "neq"
	CompareNil                CompareOperator = "nil"
	CompareLike               CompareOperator = "lyk"
	CompareIn                 CompareOperator = "in"
	CompareAny                CompareOperator = "any"
	CompareBetween            CompareOperator = "bt"
	CompareExists             CompareOperator = "exz"
	CompareRaw                CompareOperator = "-"
)

func (op CompareOperator) Eval(path []string, argCount int) string {
	field := formatPath(path)
	switch op {
	case CompareEqual:
		return field + " = " + ARG_PLACEHOLDER
	case CompareGreaterThan:
		return field + " > " + ARG_PLACEHOLDER
	case CompareLessThan:
		return field + " < " + ARG_PLACEHOLDER
	case CompareGreaterThanOrEqual:
		return field + " >= " + ARG_PLACEHOLDER
	case CompareLessThanOrEqual:
		return field + " <= " + ARG_PLACEHOLDER
	case CompareNot, CompareNotEqual:
		return field + " != " + ARG_PLACEHOLDER
	case CompareLike:
		return field + " " + SELECT_LIKE + " " + ARG_PLACEHOLDER
	case CompareIn:
		args := make([]string, argCount)
		for i := 0; i < argCount; i++ {
			args[i] = ARG_PLACEHOLDER
		}
		return field + " in (" + strings.Join(args, ",") + ")"
	case CompareAny:
		args := make([]string, argCount)
		for i := 0; i < argCount; i++ {
			args[i] = ARG_PLACEHOLDER
		}
		return field + " = any(" + strings.Join(args, ",") + ")"
	case CompareBetween:
		return field + " between " + fmt.Sprintf("%s and %s", ARG_PLACEHOLDER, ARG_PLACEHOLDER)
	case CompareExists:
		args := make([]string, argCount)
		for i := 0; i < argCount; i++ {
			args[i] = ARG_PLACEHOLDER
		}
		queryString := field
		if len(args) == 0 {
			return "exists (" + queryString + ")"
		}
		return "exists (" + strings.Replace(queryString, "<?>", strings.Join(args, ","), 1) + ")"
	default:
		return field + " " + string(op) + " " + ARG_PLACEHOLDER
	}
}

func formatPath(path []string) string {
	if len(path) == 0 {
		return ""
	}
	if len(path) == 1 {
		return path[0]
	}
	var sb strings.Builder
	sb.WriteString("(")
	sb.WriteString(path[0])
	for i := 1; i < len(path); i++ {
		if i == len(path)-1 {
			sb.WriteString(" ->> ")
		} else {
			sb.WriteString(" -> ")
		}
		sb.WriteString(fmt.Sprintf("'%s'", path[i]))
	}
	sb.WriteString(")")
	return sb.String()
}

type Constraint string

const (
	WhereAnd Constraint = "and"
	WhereOr  Constraint = "or"
)

var (
	ARG_PLACEHOLDER = "?"
)

type Condition func(*WhereCriteria)

func doLeafCompare(op CompareOperator, path []string, value interface{}) Condition {
	return func(cr *WhereCriteria) {
		wc := &WhereCriteria{
			operator: op,
			path:     path,
			value:    value,
			leaf:     true,
		}
		if value == nil {
			wc.null = true
		}
		cr.Append(cr.constraint, wc)
	}
}

func doLeafNullCompare(null bool, fields ...string) Condition {
	return func(cr *WhereCriteria) {
		if len(fields) == 0 {
			return
		}
		for i := range fields {
			cr.Append(cr.constraint, &WhereCriteria{
				path: []string{fields[i]},
				null: null,
				leaf: true,
			})
		}
	}
}

func doNode(constraint Constraint, conditions ...Condition) Condition {
	return func(cr *WhereCriteria) {
		if len(conditions) == 0 {
			return
		}
		c := &WhereCriteria{constraint: constraint}
		for i := range conditions {
			conditions[i](c)
		}
		cr.Append(constraint, c)
	}
}

func Null(fields ...string) Condition    { return Condition(doLeafNullCompare(true, fields...)) }
func NotNull(fields ...string) Condition { return Condition(doLeafNullCompare(false, fields...)) }
func Equal(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareEqual, []string{field}, value))
}
func Like(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareLike, []string{field}, value))
}
func NotEqual(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareNotEqual, []string{field}, value))
}
func LessThan(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareLessThan, []string{field}, value))
}
func GreaterThan(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareGreaterThan, []string{field}, value))
}
func LessThanOrEqual(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareLessThanOrEqual, []string{field}, value))
}
func GreaterThanOrEqual(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareGreaterThanOrEqual, []string{field}, value))
}
func In[T any](field string, values ...T) Condition {
	return Condition(doLeafCompare(CompareIn, []string{field}, values))
}
func Any[T any](field string, values ...T) Condition {
	return Condition(doLeafCompare(CompareAny, []string{field}, values))
}
func Between[T any](field string, low, high T) Condition {
	return Condition(doLeafCompare(CompareBetween, []string{field}, []T{low, high}))
}
func Exists[T any](subQuery string, args ...T) Condition {
	return Condition(doLeafCompare(CompareExists, []string{subQuery}, args))
}

func And(conditions ...Condition) Condition { return doNode(WhereAnd, conditions...) }
func Or(conditions ...Condition) Condition  { return doNode(WhereOr, conditions...) }

func UrlFieldParams(p URLSearchParam) Condition {
	conditions := make([]Condition, 0)
	for _, cond := range p.GetConditions() {
		conditions = append(conditions, doLeafCompare(cond.Operator, cond.DBPath, cond.Value))
	}
	return And(conditions...)
}

type WhereCriteria struct {
	constraint Constraint
	operator   CompareOperator
	path       []string
	value      interface{}
	null       bool
	leaf       bool

	criteria map[Constraint][]*WhereCriteria
}

func (wc *WhereCriteria) Append(cs Constraint, cr *WhereCriteria) {
	if len(wc.criteria) == 0 {
		wc.criteria = make(map[Constraint][]*WhereCriteria)
	}
	if len(wc.criteria[cs]) == 0 {
		wc.criteria[cs] = make([]*WhereCriteria, 0)
	}
	wc.criteria[cs] = append(wc.criteria[cs], cr)
}

func (wc *WhereCriteria) Build(finalise bool) (sb strings.Builder, args []interface{}) {
	if !wc.leaf {
		if len(wc.criteria) == 0 {
			return
		}
		args = make([]interface{}, 0)
		queries0 := make([]string, 0)
		for _, cs := range []Constraint{WhereOr, WhereAnd} {
			if _, ok := wc.criteria[cs]; !ok {
				continue
			}
			if len(wc.criteria[cs]) == 0 {
				continue
			}
			queries1 := make([]string, 0)
			for i := range wc.criteria[cs] {
				if wc.criteria[cs][i] == nil {
					continue
				}
				sub_sb, ar := wc.criteria[cs][i].Build(false)
				if sub_sb.Len() == 0 {
					continue
				}
				queries1 = append(queries1, sub_sb.String())
				args = append(args, ar...)
			}
			switch len(queries1) {
			case 0:
				continue
			case 1:
				queries0 = append(queries0, queries1...)
			default:
				queries0 = append(queries0, "("+strings.Join(queries1, fmt.Sprintf(") %s (", cs))+")")
			}
		}
		var query string
		switch len(queries0) {
		case 0:
			return
		case 1:
			query = queries0[0]
		default:
			query = "(" + strings.Join(queries0, fmt.Sprintf(") %s (", wc.constraint)) + ")"
		}
		if finalise {
			finalQuery := query
			for i := range args {
				finalQuery = strings.Replace(finalQuery, ARG_PLACEHOLDER, fmt.Sprintf("$%d", i+1), 1)
			}
			sb.WriteString(finalQuery)
		} else {
			sb.WriteString(query)
		}
		return
	}
	if wc.value == nil {
		field := formatPath(wc.path)
		if !wc.null {
			sb.WriteString(fmt.Sprintf("%s is not null", field))
		} else {
			sb.WriteString(fmt.Sprintf("%s is null", field))
		}
		return
	}
	switch wc.operator {
	case CompareIn, CompareBetween:
		utils.StructCopy(wc.value, &args)
	default:
		args = []interface{}{wc.value}
	}
	sb.WriteString(wc.operator.Eval(wc.path, len(args)))
	return
}
