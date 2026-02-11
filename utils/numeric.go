package utils

import (
	"fmt"
	"math"

	"math/rand/v2"

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

func Max[N NumericType](nums ...N) (out N) {
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

func Min[N NumericType](nums ...N) (out N) {
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

func Round[N float32 | float64](precision int, n N) N {
	pow := math.Pow10(precision)
	return N(math.Round(float64(n)*pow) / pow)
}

func FormatMoney(currency string, amount float64, precision int) string {
	return fmt.Sprintf("%s %s", currency, cashPrinter.Sprintf("%.2f", Round(precision, amount)))
}

var Number numUtils

type numUtils struct {
}

func (numUtils) Random(length int) int64 {
	if length <= 0 || length > 18 { // int64 can reliably store up to 18 digits
		fmt.Println("Length must be between 1 and 18 for int64")
		return 0
	}

	// Calculate the minimum and maximum values for the given length
	min := int64(math.Pow10(length - 1))
	max := int64(math.Pow10(length)) - 1 // Inclusive maximum

	// rand.Int64N returns a value in [0, max-min+1), so we add min to the result
	// The range for Int64N should be max-min+1
	return min + rand.Int64N(max-min+1)
}
