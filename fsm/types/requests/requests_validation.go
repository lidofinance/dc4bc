package requests

import "errors"

func (r *DefaultRequest) Validate() error {
	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}
