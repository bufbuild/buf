// Package appnetrc contains functionality to work with netrc.
package appnetrc

import (
	"os"
	"path/filepath"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/multierr"
)

// Machine is a machine.
type Machine interface {
	// Empty for default machine.
	Name() string
	Login() string
	Password() string
	Account() string
}

// GetMachineForName returns the Machine for the given name.
//
// Returns nil if no such Machine.
func GetMachineForName(envContainer app.EnvContainer, name string) (_ Machine, retErr error) {
	filePath, err := getFilePath(envContainer)
	if err != nil {
		return nil, err
	}
	return getMachineForNameAndFilePath(name, filePath)
}

func getFilePath(envContainer app.EnvContainer) (string, error) {
	if netrcFilePath := envContainer.Env("NETRC"); netrcFilePath != "" {
		return netrcFilePath, nil
	}
	homeDirPath, err := app.HomeDirPath(envContainer)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDirPath, netrcFilename), nil
}

func getMachineForNameAndFilePath(name string, filePath string) (_ Machine, retErr error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	netrc, err := netrc.Parse(file)
	if err != nil {
		return nil, err
	}
	netrcMachine := netrc.FindMachine(name)
	if netrcMachine == nil {
		return nil, nil
	}
	return newMachine(
		netrcMachine.Name,
		netrcMachine.Login,
		netrcMachine.Password,
		netrcMachine.Account,
	), nil
}
