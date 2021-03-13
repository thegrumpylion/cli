package cli

import "fmt"

var ErrInvalidFlag = func(flg string) error { return fmt.Errorf("invalid flag: %s", flg) }
var ErrInvalidValue = func(val, flg string) error { return fmt.Errorf("invalid value: %s for flag: %s", val, flg) }

type ErrCommandNotFound struct {
	Command string
}

func (e ErrCommandNotFound) Error() string {
	return fmt.Sprintf("command not found: %s", e.Command)
}

type ErrNoSuchFlag struct {
	Flag string
}

func (e ErrNoSuchFlag) Error() string {
	return fmt.Sprintf("no such flag: %s", e.Flag)
}
