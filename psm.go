package cli

import (
	"encoding"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/kballard/go-shellquote"
)

func newStateMachine(p *Parser) *stateMachine {
	sm := &stateMachine{
		p:         p,
		expectCmd: true,
	}
	if p.globalsEnabled {
		sm.globals = newFlagSet()
	}
	return sm
}

type stateMachine struct {
	p           *Parser
	globals     *flagSet
	curArg      *argument
	curCmd      *command
	curPos      int
	allPos      bool
	execList    []interface{}
	isComp      bool
	completeOut io.Writer
	isLast      bool
	expectCmd   bool
}

type Token int

const (
	VAL Token = iota
	CMD
	FLAG
	COMPFLAG
	ALLPOS
)

type StateFunc func(s string, t Token) (StateFunc, error)

func (sm *stateMachine) SetCurrentCmd(c *command) {
	sm.curCmd = c
	if sm.p.globalsEnabled {
		for _, a := range sm.curCmd.AllFlags() {
			if a.global {
				sm.globals.Add(a)
			}
		}
	}
	// add subcommand to execution list
	sm.execList = append(sm.execList, sm.curCmd.path.Get())
}

func (sm *stateMachine) Run(args []string) (err error) {

	isComp := isCompletion()
	if isComp {
		sm.isComp = true
		args, err = parseCompletion(args)
		if err != nil {
			os.Exit(0)
		}
	}

	c, err := sm.p.findRootCommand(args[0])
	if err != nil {
		if isComp {
			os.Exit(0)
		}
		return err
	}
	sm.SetCurrentCmd(c)

	args = args[1:]
	state := sm.entryState
	var t, lt Token

	for i, a := range args {
		lt = t
		t = sm.tokenType(a)
		if sm.allPos {
			t = VAL
		}
		if sm.isComp && i == len(args)-1 {
			break
		}
		state, err = state(a, t)
		if err != nil {
			if isComp {
				os.Exit(0)
			}
			return err
		}
	}
	// t        | tl   | isBool | action
	// --------------------------------------------
	// VAL      | FLAG | FALSE  | compArg(LV)
	// VAL      | FLAG | TRUE   | compCmd(LV)
	// CMD      | -    | -      | compCmd(LV)
	// FLAG     | -    | -      | compCmd(LV)
	// COMPFLAG | -    | -      | compArg(split(LV))
	// ALLPOS   | -    | -      | compCmd(LV)
	if sm.isComp {
		if sm.allPos {
			os.Exit(0)
		}
		completer := sm.curCmd.Complete
		val := args[len(args)-1]
		switch t {
		case COMPFLAG:
			_, val = splitCompositeFlag(val)
			completer = sm.curArg.Complete
		case VAL:
			if lt == FLAG && !sm.curArg.IsBool() {
				completer = sm.curArg.Complete
			}
		}
		for _, v := range completer(val) {
			fmt.Fprintln(sm.p.completeOut, v)
		}
		os.Exit(0)
	}
	return nil
}

func (sm *stateMachine) entryState(s string, t Token) (StateFunc, error) {
	sm.expectCmd = true
	switch t {
	case VAL:
		return sm.posArgState(s, t)
	case CMD:
		return sm.cmdState(s, t)
	case FLAG:
		return sm.flagState(s, t)
	case COMPFLAG:
		return sm.compositFlagState(s, t)
	case ALLPOS:
		sm.allPos = true
		return sm.entryState, nil
	default:
		return nil, fmt.Errorf("unknown token: %d", t)
	}
}

func (sm *stateMachine) cmdState(s string, t Token) (StateFunc, error) {
	if t != CMD {
		return nil, fmt.Errorf("unexpected token: %d at cmdState", t)
	}
	cc, ok := sm.curCmd.subcmd[s]
	if !ok {
		return nil, ErrCommandNotFound{s}
	}
	sm.SetCurrentCmd(cc)
	return sm.entryState, nil
}

func (sm *stateMachine) posArgState(s string, t Token) (StateFunc, error) {
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at posArgState", t)
	}
	if sm.curPos == len(sm.curCmd.positionals) {
		return nil, fmt.Errorf("too many positional arguments")
	}
	a := sm.curCmd.positionals[sm.curPos]
	sm.curPos++
	if a.isSlice {
		return sm.sliceValueState(s, t)
	}
	return sm.valueState(s, t)
}

func (sm *stateMachine) valueState(s string, t Token) (StateFunc, error) {
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at valueState", t)
	}
	a := sm.curArg
	if a.enum {
		em := sm.p.enums[a.typ]
		if err := a.SetValue(em[strings.ToLower(s)]); err != nil {
			return nil, err
		}
		return sm.entryState, nil
	}
	if tum, ok := a.path.Get().(encoding.TextUnmarshaler); ok {
		if err := tum.UnmarshalText([]byte(s)); err != nil {
			return nil, err
		}
		return sm.entryState, nil
	}
	if err := sm.curArg.SetScalarValue(s); err != nil {
		return nil, err
	}
	return sm.entryState, nil
}
func (sm *stateMachine) sliceValueState(s string, t Token) (StateFunc, error) {
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at sliceValueState", t)
	}
	a := sm.curArg
	if err := a.Append(s); err != nil {
		return nil, err
	}
	if a.separate {
		return sm.entryState, nil
	}
	return sm.sliceValueState, nil
}

func (sm *stateMachine) flagState(s string, t Token) (StateFunc, error) {
	if t != FLAG {
		return nil, fmt.Errorf("unexpected token: %d at flagState", t)
	}
	if sm.p.isHelp(s) {
		// handle help
	}
	if sm.p.isVersion(s) {
		// handle version
	}
	a := sm.curCmd.GetFlag(s)
	if a == nil {
		if sm.p.globalsEnabled {
			if a = sm.globals.Get(s); a == nil {
				return nil, ErrNoSuchFlag{s}
			}
		} else {
			return nil, ErrNoSuchFlag{s}
		}
	}
	sm.curArg = a
	if a.IsBool() {
		return sm.valueState("true", VAL)
	}
	sm.expectCmd = false
	if a.isSlice {
		return sm.sliceValueState, nil
	}
	return sm.valueState, nil
}

func (sm *stateMachine) compositFlagState(s string, t Token) (StateFunc, error) {
	if t != COMPFLAG {
		return nil, fmt.Errorf("unexpected token: %d at compositFlagState", t)
	}
	i := strings.Index(s, "=")
	flg := s[:i]
	val := s[i+1:]
	a := sm.curCmd.GetFlag(flg)
	if a == nil {
		if sm.p.globalsEnabled {
			fmt.Println("glob", sm.globals.All())
			if a = sm.globals.Get(s); a == nil {
				return nil, ErrNoSuchFlag{s}
			}
		} else {
			return nil, ErrNoSuchFlag{s}
		}
	}
	sm.curArg = a
	if a.isSlice {
		if !a.separate {
			return nil, fmt.Errorf("slice flag must be separated to use composite flag")
		}
		return sm.sliceValueState(s, VAL)
	}
	return sm.valueState(val, VAL)
}

func (sm *stateMachine) tokenType(s string) Token {
	if isFlag(s) {
		if i := strings.Index(s, "="); i != -1 {
			return COMPFLAG
		}
		return FLAG
	}
	if s == "--" {
		return ALLPOS
	}
	if sm.curCmd.subcmd != nil && sm.expectCmd {
		return CMD
	}
	return VAL
}

func parseCompletion(args []string) ([]string, error) {
	line := os.Getenv("COMP_LINE")
	pointS := os.Getenv("COMP_POINT")
	point, err := strconv.Atoi(pointS)
	if err != nil {
		return nil, err
	}
	if len(line) > point {
		line = line[:point]
	}
	wrds, err := shellquote.Split(line)
	if err != nil {
		return nil, err
	}
	isLastSpace := line[len(line)-1] == ' '
	if isLastSpace {
		wrds = append(wrds, "")
	}
	return wrds, nil
}
