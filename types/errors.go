package types

import "errors"

var (
	ErrExecHasDependency = errors.New("current execution has unfinished dependency")
	ErrInvalidExecID     = errors.New("invalid execution id")
	ErrNoFlow            = errors.New("flow not exists")
)
