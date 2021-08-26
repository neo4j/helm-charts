package helpers

import "github.com/hashicorp/go-multierror"

func CombineErrors(firstOrNil error, second error) error {
	if firstOrNil == nil {
		firstOrNil = second
	} else {
		firstOrNil = multierror.Append(firstOrNil, second)
	}
	return firstOrNil
}
