package api

import "fmt"

type ErrCreateOpenapiSchema struct {
	name string
	err  error
}

func (e ErrCreateOpenapiSchema) Error() string {
	return fmt.Sprintf("failed to get schema for check %s: %v", e.name, e.err)
}
