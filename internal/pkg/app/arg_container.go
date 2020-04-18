package app

type argContainer struct {
	values []string
}

func newArgContainer(s []string) *argContainer {
	values := make([]string, len(s))
	copy(values, s)
	return &argContainer{
		values: values,
	}
}

func (a *argContainer) NumArgs() int {
	return len(a.values)
}

func (a *argContainer) Arg(i int) string {
	return a.values[i]
}
