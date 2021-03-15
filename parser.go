package cnc

import (
	"encoding"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kballard/go-shellquote"
)

func newParser(cli *CLI) *parser {
	p := &parser{
		cli:       cli,
		expectCmd: true,
	}
	if cli.options.globalsEnabled {
		p.globals = newFlagSet()
	}
	return p
}

type parser struct {
	cli       *CLI
	globals   *flagSet
	curArg    *argument
	curCmd    *command
	curPos    int
	allPos    bool
	execList  []interface{}
	isComp    bool
	expectCmd bool
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

func (p *parser) Run(args []string) (err error) {

	isComp := isCompletion()
	if isComp {
		p.isComp = true
		args, err = parseCompletion(args)
		if err != nil {
			os.Exit(0)
		}
	}

	c, err := p.cli.findRootCommand(args[0])
	if err != nil {
		if isComp {
			os.Exit(0)
		}
		return err
	}
	p.setCurrentCmd(c)

	args = args[1:]
	state := p.entryState
	var t, lt Token

	for i, a := range args {
		lt = t
		t = p.tokenType(a)
		if p.allPos {
			t = VAL
		}
		if p.isComp && i == len(args)-1 {
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
	if p.isComp {
		if p.allPos {
			os.Exit(0)
		}
		var completer Completer = NewFuncCmpleter(p.currentCmd().CompleteSubcommands)
		val := args[len(args)-1]
		switch t {
		case COMPFLAG:
			fg, vl := splitCompositeFlag(val)
			flg := p.currentCmd().GetFlag(fg)
			if flg != nil {
				completer = flg
				val = vl
			}
		case VAL:
			if lt == FLAG && !p.currentArg().IsBool() {
				completer = p.currentArg()
			}
		case FLAG, ALLPOS:
			completer = NewFuncCmpleter(p.currentCmd().CompleteFlags)
		}
		if completer != nil {
			for _, v := range completer.Complete(val) {
				fmt.Fprintln(p.cli.completeOut, v)
			}
		}
		os.Exit(0)
	}
	return nil
}

func (p *parser) ExecList() []interface{} {
	return p.execList
}

func (p *parser) setCurrentCmd(c *command) {
	p.curCmd = c
	if p.cli.options.globalsEnabled {
		for _, a := range p.currentCmd().Flags() {
			if a.global {
				p.globals.Add(a)
			}
		}
	}
	// add subcommand to execution list
	p.execList = append(p.execList, p.currentCmd().path.Get())
}

func (p *parser) currentCmd() *command {
	return p.curCmd
}

func (p *parser) setCurrentArg(a *argument) {
	p.curArg = a
}

func (p *parser) currentArg() *argument {
	return p.curArg
}

func (p *parser) entryState(s string, t Token) (StateFunc, error) {
	fmt.Println("entryState", s, t)
	p.expectCmd = true
	switch t {
	case VAL:
		return p.posArgState(s, t)
	case CMD:
		return p.cmdState(s, t)
	case FLAG:
		return p.flagState(s, t)
	case COMPFLAG:
		return p.compositFlagState(s, t)
	case ALLPOS:
		p.allPos = true
		return p.entryState, nil
	default:
		return nil, fmt.Errorf("unknown token: %d", t)
	}
}

func (p *parser) cmdState(s string, t Token) (StateFunc, error) {
	fmt.Println("cmdState", s, t)
	if t != CMD {
		return nil, fmt.Errorf("unexpected token: %d at cmdState", t)
	}
	cc, ok := p.currentCmd().LookupSubcommand(s)
	if !ok {
		return nil, ErrCommandNotFound{s}
	}
	p.setCurrentCmd(cc)
	return p.entryState, nil
}

func (p *parser) posArgState(s string, t Token) (StateFunc, error) {
	fmt.Println("posArgState", s, t)
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at posArgState", t)
	}
	if p.curPos == len(p.currentCmd().positionals) {
		return nil, fmt.Errorf("too many positional arguments")
	}
	a := p.currentCmd().positionals[p.curPos]
	p.setCurrentArg(a)
	p.curPos++
	if a.isSlice {
		return p.sliceValueState(s, t)
	}
	return p.valueState(s, t)
}

func (p *parser) valueState(s string, t Token) (StateFunc, error) {
	fmt.Println("valueState", s, t)
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at valueState", t)
	}
	a := p.currentArg()
	if a.enum != nil {
		if err := a.SetValue(a.enum.Value(s)); err != nil {
			return nil, err
		}
		return p.entryState, nil
	}
	if tum, ok := a.path.Get().(encoding.TextUnmarshaler); ok {
		if err := tum.UnmarshalText([]byte(s)); err != nil {
			return nil, err
		}
		return p.entryState, nil
	}
	if err := p.currentArg().SetScalarValue(s); err != nil {
		return nil, err
	}
	return p.entryState, nil
}
func (p *parser) sliceValueState(s string, t Token) (StateFunc, error) {
	fmt.Println("sliceValueState", s, t)
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at sliceValueState", t)
	}
	a := p.currentArg()
	if err := a.Append(s); err != nil {
		return nil, err
	}
	return p.entryState, nil
}

func (p *parser) flagState(s string, t Token) (StateFunc, error) {
	fmt.Println("flagState", s, t)
	if t != FLAG {
		return nil, fmt.Errorf("unexpected token: %d at flagState", t)
	}
	if p.cli.isHelp(s) {
		p.currentCmd().Usage(p.cli.helpOut)
		os.Exit(0)
	}
	if p.cli.isVersion(s) {
		// handle version
	}
	a := p.currentCmd().GetFlag(s)
	if a == nil {
		if p.cli.options.globalsEnabled {
			if a = p.globals.Get(s); a == nil {
				return nil, ErrNoSuchFlag{s}
			}
		} else {
			return nil, ErrNoSuchFlag{s}
		}
	}
	p.setCurrentArg(a)
	if a.IsBool() {
		return p.valueState("true", VAL)
	}
	p.expectCmd = false
	if a.isSlice {
		return p.sliceValueState, nil
	}
	return p.valueState, nil
}

func (p *parser) compositFlagState(s string, t Token) (StateFunc, error) {
	fmt.Println("compositFlagState", s, t)
	if t != COMPFLAG {
		return nil, fmt.Errorf("unexpected token: %d at compositFlagState", t)
	}
	i := strings.Index(s, "=")
	flg := s[:i]
	val := s[i+1:]
	a := p.currentCmd().GetFlag(flg)
	if a == nil {
		if p.cli.options.globalsEnabled {
			fmt.Println("glob", p.globals.All())
			if a = p.globals.Get(s); a == nil {
				return nil, ErrNoSuchFlag{s}
			}
		} else {
			return nil, ErrNoSuchFlag{s}
		}
	}
	p.setCurrentArg(a)
	if a.isSlice {
		return p.sliceValueState(s, VAL)
	}
	return p.valueState(val, VAL)
}

func (p *parser) tokenType(s string) Token {
	if isFlag(s) {
		if i := strings.Index(s, "="); i != -1 {
			return COMPFLAG
		}
		return FLAG
	}
	if s == "--" {
		return ALLPOS
	}
	if len(p.currentCmd().subcmdsMap) != 0 && p.expectCmd {
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
