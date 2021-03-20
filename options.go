package cli

type Separator byte

const (
	SeparatorSpace  Separator = ' '
	SeparatorEquals Separator = '='
)

type cliOptions struct {
	tags           StructTags
	globalsEnabled bool
	argCase        Case
	envCase        Case
	cmdCase        Case
	argSplicer     Splicer
	envSplicer     Splicer
	helpLong       string
	helpShort      string
	versionLong    string
	versionShort   string
	strategy       OnErrorStrategy
	separator      Separator
	cmdColSize     uint
	flagColSize    uint
}

// Option option type for Parser
type Option func(o *cliOptions)

// WithArgCase set the arg case. default is CaseCamelLower
func WithArgCase(c Case) Option {
	return func(o *cliOptions) {
		o.argCase = c
	}
}

// WithEnvCase set the env case. default is CaseSnakeUpper
func WithEnvCase(c Case) Option {
	return func(o *cliOptions) {
		o.envCase = c
	}
}

// WithCmdCase set the cmd case. default is CaseLower
func WithCmdCase(c Case) Option {
	return func(o *cliOptions) {
		o.cmdCase = c
	}
}

// WithArgSplicer set the arg splicer
func WithArgSplicer(s Splicer) Option {
	return func(o *cliOptions) {
		o.argSplicer = s
	}
}

// WithEnvSplicer set the env splicer
func WithEnvSplicer(s Splicer) Option {
	return func(o *cliOptions) {
		o.envSplicer = s
	}
}

// WithOnErrorStrategy sets the execution strategy for handling errors
func WithOnErrorStrategy(str OnErrorStrategy) Option {
	return func(o *cliOptions) {
		o.strategy = str
	}
}

// WithGlobalArgsEnabled enable global argumets
func WithGlobalArgsEnabled() Option {
	return func(o *cliOptions) {
		o.globalsEnabled = true
	}
}

// WithStructTags sets the struct tags to be used by this parser
func WithStructTags(tags StructTags) Option {
	return func(o *cliOptions) {
		o.tags = tags
	}
}

// WithHelpFlags sets the help flags. Default --help,-h
func WithHelpFlags(long, short string) Option {
	return func(o *cliOptions) {
		o.helpLong = long
		o.helpShort = short
	}
}

// WithVersionFlags sets the version flags. Default --version
func WithVersionFlags(long, short string) Option {
	return func(o *cliOptions) {
		o.versionLong = long
		o.versionShort = short
	}
}

// WithSeparator sets the flag separator charachter for help and completion
func WithSeparator(sep Separator) Option {
	return func(o *cliOptions) {
		o.separator = sep
	}
}
