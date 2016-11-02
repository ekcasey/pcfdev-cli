package ssh

type ExitError struct {
	Err error
	StatusCode int
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}
