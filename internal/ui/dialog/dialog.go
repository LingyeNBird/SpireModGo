package dialog

import (
	"errors"
	"fmt"
)

var ErrCancelled = errors.New("file dialog cancelled")

type FileDialogError struct {
	Err    error
	Detail string
}

func (e *FileDialogError) Error() string {
	if e == nil {
		return ""
	}
	if e.Detail == "" {
		return fmt.Sprintf("run file dialog: %v", e.Err)
	}
	return fmt.Sprintf("run file dialog: %v (%s)", e.Err, e.Detail)
}

func (e *FileDialogError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
