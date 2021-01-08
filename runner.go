package cli

import (
	"context"
	"reflect"
)

// OnErrorStrategy defines how errors are handled on execution
type OnErrorStrategy uint

const (
	// OnErrorBreak halt execution and return the error immediately
	OnErrorBreak OnErrorStrategy = iota
	// OnErrorPostRunners execute post runners in stack but break if post runner returns error.
	// LastErrorFromContext can be used to retrieve the error
	OnErrorPostRunners
	// OnErrorPostRunnersContinue execute post runners in stack ignoring errors. LastErrorFromContext
	// can be used to retrieve any error
	OnErrorPostRunnersContinue
	// OnErrorContinue ignore errors. LastErrorFromContext can be used to retrieve any error.
	OnErrorContinue
)

type lastErrorKey struct{}

// LastErrorFromContext get the last error in case the execution continues on errors
func LastErrorFromContext(ctx context.Context) error {
	return ctx.Value(lastErrorKey{}).(error)
}

// Runner interface
type Runner interface {
	Run(ctx context.Context) error
}

// PreRunner interface
type PreRunner interface {
	PreRun(ctx context.Context) error
}

// PersistentPreRunner interface
type PersistentPreRunner interface {
	PersistentPreRun(ctx context.Context) error
}

// PostRunner interface
type PostRunner interface {
	PostRun(ctx context.Context) error
}

// PersistentPostRunner interface
type PersistentPostRunner interface {
	PersistentPostRun(ctx context.Context) error
}

var runnerType = reflect.TypeOf((*Runner)(nil)).Elem()
var preRunnerType = reflect.TypeOf((*PreRunner)(nil)).Elem()
var persistentPreRunnerType = reflect.TypeOf((*PersistentPreRunner)(nil)).Elem()
var postRunnerType = reflect.TypeOf((*PostRunner)(nil)).Elem()
var persistentPostRunnerType = reflect.TypeOf((*PersistentPostRunner)(nil)).Elem()

func isRunner(t reflect.Type) bool {
	return t.Implements(runnerType) || t.Implements(preRunnerType) ||
		t.Implements(persistentPreRunnerType) || t.Implements(postRunnerType) ||
		t.Implements(persistentPostRunnerType)
}
