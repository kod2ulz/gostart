package utils

import (
	"fmt"
	"math"

	"golang.org/x/text/language"
	msg "golang.org/x/text/message"
)

var cashPrinter *msg.Printer

func init() {
	cashPrinter = msg.NewPrinter(language.English)
}

type NumericType interface {
	int | int8 | int16 | int32 | int64 
}

func Max[N NumericType](nums...N) (out N) {
	if len(nums) == 0 {
		return
	}
	out = nums[0]
	for i := range nums {
		if nums[i] > out {
			out = nums[i]
		}
	}
	return
}

func Min[N NumericType](nums...N) (out N) {
	if len(nums) == 0 {
		return
	}
	out = nums[0]
	for i := range nums {
		if nums[i] < out {
			out = nums[i]
		}
	}
	return
}

func Round[N float32|float64] (precision int, n N) N {
	pow := math.Pow10(precision)
	return N(math.Round(float64(n)*pow)/pow)
}

func FormatMoney(currency string, amount float64, precision int) string {
	return fmt.Sprintf("%s %s", currency, cashPrinter.Sprintf("%.2f", Round(precision, amount)))
}