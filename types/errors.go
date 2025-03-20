package types

import "errors"

var (
	ErrExecHasDependency   = errors.New("current execution has unfinished dependency")
	ErrInvalidExecID       = errors.New("invalid execution id")
	ErrNoFlow              = errors.New("flow not exists")
	ErrExecResultNotExists = errors.New("execution result not exists")
	ErrExecResultExists    = errors.New("execution result exists")
)
