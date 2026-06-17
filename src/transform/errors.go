package transform

type transformError struct{}

func (e *transformError) Error() string { return "unrecognized rule format" }

var errUnrecognized = &transformError{}