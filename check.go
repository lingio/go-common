package common

import "errors"

func CheckNil(params... interface{}) error {
	for i, p := range params {
		if p == nil {
			return errors.New("Unexpected null pointer at param " + string(i))
		}
	}
	return nil
}

