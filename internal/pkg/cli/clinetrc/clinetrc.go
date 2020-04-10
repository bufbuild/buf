// Package clinetrc contains functionality to work with netrc.
package clinetrc

import (
	"os"
	"path/filepath"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/bufbuild/buf/internal/pkg/cli/clios"
	"go.uber.org/multierr"
)

const netrcEnvKey = "NETRC"

// Machine is a machine.
type Machine struct {
	// Will be empty on default
	Name     string
	Login    string
	Password string
	Account  string
}

// GetMachineByName gets the Machine for the given name.
//
// Returns nil if no such Machine.
func GetMachineByName(name string, getenv func(string) string) (_ *Machine, retErr error) {
	netrcFilePath, err := getNetrcFilePath(getenv)
	if err != nil {
		return nil, err
	}
	netrcFile, err := os.Open(netrcFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, netrcFile.Close())
	}()
	netrc, err := netrc.Parse(netrcFile)
	if err != nil {
		return nil, err
	}
	netrcMachine := netrc.FindMachine(name)
	if netrcMachine == nil {
		return nil, nil
	}
	return &Machine{
		Name:     netrcMachine.Name,
		Login:    netrcMachine.Login,
		Password: netrcMachine.Password,
		Account:  netrcMachine.Account,
	}, nil
}

func getNetrcFilePath(getenv func(string) string) (string, error) {
	if netrcFilePath := getenv(netrcEnvKey); netrcFilePath != "" {
		return netrcFilePath, nil
	}
	homeDirPath, err := clios.Home(getenv)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDirPath, netrcFilename), nil
}
