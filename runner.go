package cli

import (
	"context"
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

// Execute the chain of commands in default parser
func Execute(ctx context.Context) error {
	return defaultCLI.Execute(ctx)
}

// Execute the chain of commands
func (cli *CLI) Execute(ctx context.Context) error {

	var err error
	lastCmd := len(cli.execList) - 1
	pPostRunners := []PersistentPostRunner{}

	for i, inf := range cli.execList {
		// PersistentPostRun pushed on a stack to run in a reverse order
		if rnr, ok := inf.(PersistentPostRunner); ok {
			pPostRunners = append([]PersistentPostRunner{rnr}, pPostRunners...)
		}
		// PersistentPreRun
		if rnr, ok := inf.(PersistentPreRunner); ok {
			err = rnr.PersistentPreRun(ctx)
			if err != nil {
				if !(cli.options.strategy == OnErrorContinue) {
					break
				}
				ctx = context.WithValue(ctx, lastErrorKey{}, err)
			}
		}
		if i == lastCmd {
			// PreRun
			if rnr, ok := inf.(PreRunner); ok {
				err = rnr.PreRun(ctx)
				if err != nil {
					if !(cli.options.strategy == OnErrorContinue) {
						break
					}
					ctx = context.WithValue(ctx, lastErrorKey{}, err)
				}
			}
			// Run
			if rnr, ok := inf.(Runner); ok {
				err = rnr.Run(ctx)
				if err != nil {
					if !(cli.options.strategy == OnErrorContinue) {
						break
					}
					ctx = context.WithValue(ctx, lastErrorKey{}, err)
				}
			}
			// PostRun
			if rnr, ok := inf.(PostRunner); ok {
				err = rnr.PostRun(ctx)
				if err != nil {
					if !(cli.options.strategy == OnErrorContinue) {
						break
					}
					ctx = context.WithValue(ctx, lastErrorKey{}, err)
				}
			}
		}
	}
	// check for error and strategy
	if err != nil && cli.options.strategy == OnErrorBreak {
		return err
	}
	// PersistentPostRun
	for _, rnr := range pPostRunners {
		err = rnr.PersistentPostRun(ctx)
		if err != nil {
			if cli.options.strategy == OnErrorPostRunners {
				return err
			}
			ctx = context.WithValue(ctx, lastErrorKey{}, err)
		}
	}
	return err
}
