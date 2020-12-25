package cli

import (
	"context"
	"fmt"
	"reflect"
)

type Runner interface {
	Run(ctx context.Context, lastErr error) error
}

type PreRunner interface {
	PreRun(ctx context.Context, lastErr error) error
}

type PersistentPreRunner interface {
	PersistentPreRun(ctx context.Context, lastErr error) error
}

type PostRunner interface {
	PostRun(ctx context.Context, lastErr error) error
}

type PersistentPostRunner interface {
	PersistentPostRun(ctx context.Context, lastErr error) error
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

type BreakFlowError struct {
	e error
}

func (e *BreakFlowError) Error() string {
	return fmt.Sprintf("BRK FLOW: %v", e.e)
}

func (e *BreakFlowError) Unwrap() error {
	return e.e
}
