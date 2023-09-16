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
)

func (op CompareOperator) Eval(field string, argCount int) string {
	switch op {
	case CompareEqual:
		return field + "=" + ARG_PLACEHOLDER
	case CompareGreaterThan:
		return field + ">" + ARG_PLACEHOLDER
	case CompareLessThan:
		return field + "<" + ARG_PLACEHOLDER
	case CompareGreaterThanOrEqual:
		return field + ">=" + ARG_PLACEHOLDER
	case CompareLessThanOrEqual:
		return field + "<=" + ARG_PLACEHOLDER
	case CompareNot, CompareNotEqual:
		return field + "!=" + ARG_PLACEHOLDER
	case CompareLike:
		return field + " " + SELECT_LIKE + " " + ARG_PLACEHOLDER
	case CompareIn:
		args := make([]string, argCount)
		for i := 0; i< argCount; i++ {
			args[i] = ARG_PLACEHOLDER
		}
		return field + " in (" + strings.Join(args, ",") + ")"
	default:
		return field + " " + string(op) + " " + ARG_PLACEHOLDER
	}
}

type Constraint string

const (
	WhereAnd Constraint = "and"
	WhereOr  Constraint = "or"
)

var (
	ARG_PLACEHOLDER = "????"
)

type Condition func(*WhereCriteria)

func doLeafCompare(op CompareOperator, field string, value interface{}) Condition {
	return func(cr *WhereCriteria) {
		wc := &WhereCriteria{
			operator: op,
			field:    field,
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
				field: fields[i],
				null:  null,
				leaf:  true,
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
	return Condition(doLeafCompare(CompareEqual, field, value))
}
func Like(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareLike, field, value))
}
func NotEqual(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareNotEqual, field, value))
}
func LessThan(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareLessThan, field, value))
}
func GreaterThan(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareGreaterThan, field, value))
}
func LessThanOrEqual(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareLessThanOrEqual, field, value))
}
func GreaterThanOrEqual(field string, value interface{}) Condition {
	return Condition(doLeafCompare(CompareGreaterThanOrEqual, field, value))
}
func In[T any](field string, values ...T) Condition {
	return Condition(doLeafCompare(CompareIn, field, values))
}

func And(conditions ...Condition) Condition { return doNode(WhereAnd, conditions...) }
func Or(conditions ...Condition) Condition  { return doNode(WhereOr, conditions...) }

func UrlFieldParams(p URLSearchParam) Condition {
	conditions := make([]Condition, 0)
	if len(p.GetFieldValues()) > 0 {
		for field, value := range p.GetFieldValues() {
			conditions = append(conditions, Equal(field, value))
		}
	}
	if len(p.GetFieldNullables()) > 0 {
		for field, null := range p.GetFieldNullables() {
			if null {
				conditions = append(conditions, Null(field))
			} else {
				conditions = append(conditions, NotNull(field))
			}
		}
	}
	if len(p.GetFieldComparisons()) > 0 {
		for field, compare := range p.GetFieldComparisons() {
			for op, val := range compare {
				conditions = append(conditions, doLeafCompare(op, field, val))
			}
		}
	}
	return And(conditions...)
}

type WhereCriteria struct {
	constraint Constraint
	operator   CompareOperator
	field      string
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

func (wc *WhereCriteria) finalise(do bool, sb *strings.Builder, val string, args ...interface{}) {
	if !do || val == "" || len(args) == 0 {
		sb.WriteString(val)
		return
	}
	var out string = val
	for i := range args {
		out = strings.Replace(out, ARG_PLACEHOLDER, fmt.Sprintf("$%d", i+1), 1)
	}
	sb.WriteString(out)
}

func (wc *WhereCriteria) Build(finalise bool) (sb strings.Builder, args []interface{}) {
	if !wc.leaf {
		if len(wc.criteria) == 0 {
			return
		}
		args = make([]interface{}, 0)
		queries0 := make([]string, 0)
		for cs := range wc.criteria {
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
		switch len(queries0) {
		case 0:
			return
		case 1:
			wc.finalise(finalise, &sb, queries0[0], args...)
		default:
			wc.finalise(finalise, &sb, "("+strings.Join(queries0, fmt.Sprintf(") %s (", wc.constraint))+")", args...)
		}
		return
	}
	if wc.value == nil {
		if !wc.null {
			sb.WriteString(fmt.Sprintf("%s is not null", wc.field))
		} else {
			sb.WriteString(fmt.Sprintf("%s is null", wc.field))
		}
		return
	}
	if wc.operator == CompareIn {
		utils.StructCopy(wc.value, &args)
	} else {
		args = []interface{}{wc.value}
	}
	sb.WriteString(wc.operator.Eval(wc.field, len(args)))
	return
}
