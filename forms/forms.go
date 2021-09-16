package forms

import (
	"encoding/json"
	"fmt"
	"net/http"

	"libs.altipla.consulting/errors"
	"libs.altipla.consulting/routing"
)

type ValidationError struct {
	message string
}

func (v ValidationError) Error() string {
	return v.message
}

func Errorf(msg string, args ...interface{}) error {
	return ValidationError{fmt.Sprintf(msg, args...)}
}

type Validatable interface {
	Validate() error
}

func Read(r *http.Request, dest Validatable) error {
	if err := json.Unmarshal([]byte(r.FormValue("$value")), dest); err != nil {
		return errors.Trace(err)
	}

	if err := dest.Validate(); err != nil {
		var v ValidationError
		if errors.As(err, &v) {
			return routing.BadRequest(v.message)
		}

		return errors.Trace(err)
	}

	// TODO(ernesto): Trim all dest strings inside struct like we do with gRPC.

	return nil
}
