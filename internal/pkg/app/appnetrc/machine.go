package appnetrc

type machine struct {
	name     string
	login    string
	password string
	account  string
}

func newMachine(
	name string,
	login string,
	password string,
	account string,
) *machine {
	return &machine{
		name:     name,
		login:    login,
		password: password,
		account:  account,
	}
}

func (m *machine) Name() string {
	return m.name
}

func (m *machine) Login() string {
	return m.login
}

func (m *machine) Password() string {
	return m.password
}

func (m *machine) Account() string {
	return m.account
}
