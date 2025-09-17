package query_test

import (
	"testing"

	"github.com/kod2ulz/gostart/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQuery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Query Suite")
}

func copyDefinitions(def query.FieldDefinitions) (out query.FieldDefinitions) {
	out = query.NewDefinitions()
	for k, v := range def {
		out[k] = v
	}
	return
}

var findCondition = func(operator query.CompareOperator) func(query.ParsedCondition) bool {
	return func(pc query.ParsedCondition) bool {
		return pc.Operator == operator
	}
}
