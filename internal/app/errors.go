package app

// error categories for internal use — mapped to output.Category at the boundary.
type errCategory string

const (
	catValidation errCategory = "validation"
	catAuth       errCategory = "auth"
	catNotFound   errCategory = "not_found"
	catError      errCategory = "error"
)

type appError struct {
	cat errCategory
	msg string
	err error
}

func (e *appError) Error() string {
	if e.err != nil {
		return e.msg + ": " + e.err.Error()
	}
	return e.msg
}

func (e *appError) Unwrap() error { return e.err }

// Category returns the error's category string for output boundary mapping.
func (e *appError) Category() string { return string(e.cat) }
