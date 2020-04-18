package app

type container struct {
	EnvContainer
	StdinContainer
	StdoutContainer
	StderrContainer
	ArgContainer
}

func newContainer(
	envContainer EnvContainer,
	stdinContainer StdinContainer,
	stdoutContainer StdoutContainer,
	stderrContainer StderrContainer,
	argContainer ArgContainer,
) *container {
	return &container{
		EnvContainer:    envContainer,
		StdinContainer:  stdinContainer,
		StdoutContainer: stdoutContainer,
		StderrContainer: stderrContainer,
		ArgContainer:    argContainer,
	}
}
