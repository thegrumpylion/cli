package cli

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
	runList   []interface{}
	isComp    bool
	expectCmd bool
	debug     bool
}

type parserToken int

const (
	tokVAL parserToken = iota
	tokCMD
	tokFLAG
	tokCOMPFLAG
	tokALLPOS
)

type StateFunc func(s string, t parserToken) (StateFunc, error)

func (p *parser) Run(args []string) (err error) {

	isComp := isCompletion()
	if isComp {
		p.isComp = true
		args, err = parseCompletion(args)
		if err != nil {
			p.cli.osExit(0)
		}
	}

	c, err := p.cli.findRootCommand(args[0])
	if err != nil {
		if isComp {
			p.cli.osExit(0)
		}
		return err
	}
	p.setCurrentCmd(c)

	args = args[1:]
	state := p.entryState
	var t, lt parserToken

	for i, a := range args {
		lt = t
		t = p.tokenType(a)
		if p.allPos {
			t = tokVAL
		}
		if p.isComp && i == len(args)-1 {
			break
		}
		state, err = state(a, t)
		if err != nil {
			if isComp {
				p.cli.osExit(0)
			}
			return err
		}
	}
	if p.isComp {
		if p.allPos {
			p.cli.osExit(0)
		}
		var completer Completer
		if p.currentCmd().HasSubcommands() {
			completer = NewFuncCmpleter(p.currentCmd().CompleteSubcommands)
		} else {
			completer = NewFuncCmpleter(p.currentCmd().CompleteFlags)
		}
		val := args[len(args)-1]
		switch t {
		case tokCOMPFLAG:
			fg, vl := splitCompositeFlag(val)
			flg := p.currentCmd().GetFlag(fg)
			if flg != nil {
				completer = flg
				val = vl
			}
		case tokVAL:
			if lt == tokFLAG && !p.currentArg().IsBool() {
				completer = p.currentArg()
			}
		case tokFLAG, tokALLPOS:
			completer = NewFuncCmpleter(p.currentCmd().CompleteFlags)
		}
		if completer != nil {
			for _, v := range completer.Complete(val) {
				fmt.Fprintln(p.cli.completeOut, v)
			}
		}
		p.cli.osExit(0)
	}
	return nil
}

func (p *parser) RunList() []interface{} {
	return p.runList
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
	p.runList = append(p.runList, p.currentCmd().path.Get())
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

func (p *parser) entryState(s string, t parserToken) (StateFunc, error) {
	p.debugln("entryState", s, t)
	p.expectCmd = true
	switch t {
	case tokVAL:
		return p.posArgState(s, t)
	case tokCMD:
		return p.cmdState(s, t)
	case tokFLAG:
		return p.flagState(s, t)
	case tokCOMPFLAG:
		return p.compositFlagState(s, t)
	case tokALLPOS:
		p.allPos = true
		return p.entryState, nil
	default:
		return nil, fmt.Errorf("unknown token: %d", t)
	}
}

func (p *parser) cmdState(s string, t parserToken) (StateFunc, error) {
	p.debugln("cmdState", s, t)
	if t != tokCMD {
		return nil, fmt.Errorf("unexpected token: %d at cmdState", t)
	}
	cc, ok := p.currentCmd().LookupSubcommand(s)
	if !ok {
		return nil, ErrCommandNotFound{s}
	}
	p.setCurrentCmd(cc)
	return p.entryState, nil
}

func (p *parser) posArgState(s string, t parserToken) (StateFunc, error) {
	p.debugln("posArgState", s, t)
	if t != tokVAL {
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

func (p *parser) valueState(s string, t parserToken) (StateFunc, error) {
	p.debugln("valueState", s, t)
	if t != tokVAL {
		return nil, fmt.Errorf("unexpected token: %d at valueState", t)
	}
	a := p.currentArg()
	if tum, ok := a.path.Get().(encoding.TextUnmarshaler); ok {
		if err := tum.UnmarshalText([]byte(s)); err != nil {
			return nil, err
		}
		return p.entryState, nil
	}
	if err := p.currentArg().SetValue(s); err != nil {
		return nil, err
	}
	if p.currentCmd().HasSubcommands() {
		p.expectCmd = true
	}
	return p.entryState, nil
}
func (p *parser) sliceValueState(s string, t parserToken) (StateFunc, error) {
	p.debugln("sliceValueState", s, t)
	if t != tokVAL {
		return nil, fmt.Errorf("unexpected token: %d at sliceValueState", t)
	}
	a := p.currentArg()
	if err := a.Append(s); err != nil {
		return nil, err
	}
	return p.entryState, nil
}

func (p *parser) flagState(s string, t parserToken) (StateFunc, error) {
	p.debugln("flagState", s, t)
	if t != tokFLAG {
		return nil, fmt.Errorf("unexpected token: %d at flagState", t)
	}
	if p.cli.isHelp(s) {
		p.currentCmd().Usage(p.cli.helpOut)
		p.cli.osExit(0)
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
		return p.valueState("true", tokVAL)
	}
	p.expectCmd = false
	if a.isSlice {
		return p.sliceValueState, nil
	}
	return p.valueState, nil
}

func (p *parser) compositFlagState(s string, t parserToken) (StateFunc, error) {
	p.debugln("compositFlagState", s, t)
	if t != tokCOMPFLAG {
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
		return p.sliceValueState(s, tokVAL)
	}
	return p.valueState(val, tokVAL)
}

func (p *parser) tokenType(s string) parserToken {
	if isFlag(s) {
		if i := strings.Index(s, "="); i != -1 {
			return tokCOMPFLAG
		}
		return tokFLAG
	}
	if s == "--" {
		return tokALLPOS
	}
	if p.currentCmd().HasSubcommands() && p.expectCmd {
		return tokCMD
	}
	return tokVAL
}

func (p *parser) debugln(a ...interface{}) {
	if p.debug {
		fmt.Println(a...)
	}
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
