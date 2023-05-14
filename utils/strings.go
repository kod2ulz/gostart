package utils

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/google/uuid"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
const (
    letterIdxBits = 6                    // 6 bits to represent a letter index
    letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
    letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func init() {
	String = strUtils{
		src: rand.NewSource(time.Now().UnixNano()),
	}
}

var String strUtils

type strUtils struct {
	src rand.Source
}

func (strUtils) ToInt(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}

func (u strUtils) ToInt64(str string) int64 {
	if n, e := strconv.ParseInt(str, 10, 64); e == nil {
		return n
	}
	return 0
}

func (u strUtils) ToInt64Slice(strs []string) []int64 {
	if len(strs) == 0 {
		return []int64{}
	}
	out := make([]int64, len(strs))
	for i := range strs {
		out[i] = u.ToInt64(strs[i])
	}
	return out
}

func (u strUtils) ToFloat64(str string) float64 {
	if n, e := strconv.ParseFloat(str, 64); e == nil {
		return n
	}
	return 0
}

func (u strUtils) FromFloat64(num float64, precision int) string {
	return fmt.Sprint(Round(precision, num))
}

func (u strUtils) ToNullUUID(str string) uuid.NullUUID {
	if n, e := uuid.Parse(str); e == nil {
		return Null.UUID(n)
	}
	return uuid.NullUUID{}
}

func (u strUtils) ToUUID(str string) uuid.UUID {
	if n, e := uuid.Parse(str); e == nil {
		return n
	}
	return uuid.Nil
}

func (u strUtils) TrimPrefixes(str string, prefixes...string) (out string) {
	if len(prefixes) == 0 {
		return str
	}
	out = str 
	for i := range prefixes {
		out = strings.TrimPrefix(out, prefixes[i])
	}
	return 
}

func (u strUtils) Random(n int) (out string) {
	b := make([]byte, n)
    // A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
    for i, cache, remain := n-1, u.src.Int63(), letterIdxMax; i >= 0; {
        if remain == 0 {
            cache, remain = u.src.Int63(), letterIdxMax
        }
        if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
            b[i] = letterBytes[idx]
            i--
        }
        cache >>= letterIdxBits
        remain--
    }

    return *(*string)(unsafe.Pointer(&b))
}