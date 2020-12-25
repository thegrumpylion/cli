package cli

import "fmt"

var ErrCommandNotFound = func(cmd string) error { return fmt.Errorf("command not found: %s", cmd) }
var ErrNoSuchFlag = func(flg string) error { return fmt.Errorf("no such flag: %s", flg) }
var ErrInvalidFlag = func(flg string) error { return fmt.Errorf("invalid flag: %s", flg) }
var ErrInvalidValue = func(val, flg string) error { return fmt.Errorf("invalid value: %s for flag: %s", val, flg) }
