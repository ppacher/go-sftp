package sshfxp

import "fmt"

type FxpStatusError struct {
	Code    uint32
	Message string
}

func (f *FxpStatusError) Error() string {
	return fmt.Sprintf("%d: %s", f.Code, f.Message)
}

func IsError(x interface{}) error {
	if status, ok := x.(*Status); ok {
		if status.Error != StatusOK {
			return &FxpStatusError{
				Code:    status.Error,
				Message: status.Message,
			}
		}
	}

	return nil
}
