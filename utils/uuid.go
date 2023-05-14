package utils

import "github.com/google/uuid"

type UuidList []uuid.UUID

func (l *UuidList) Iterate(fn func(uuid.UUID) error) (err error) {
	if len(*l) == 0 {
		return
	}
	for i := range *l {
		if err = fn((*l)[i]); err != nil {
			return
		} 
	}
	return 
}