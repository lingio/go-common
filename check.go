package common

import (
	"errors"
	"fmt"
)

func CheckNil(params ...interface{}) error {
	for i, p := range params {
		if p == nil {
			return errors.New(fmt.Sprintf("Unexpected null pointer at param %d", i))
		}
	}
	return nil
}
