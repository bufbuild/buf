// Copyright 2020-2021 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package netrc contains functionality to work with netrc.
package netrc

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/multierr"
)

// Filename exposes the netrc filename based on the current operating system.
const Filename = netrcFilename

// Machine is a machine.
type Machine interface {
	// Empty for default machine.
	Name() string
	Login() string
	Password() string
	Account() string
}

// NewMachine creates a new Machine.
func NewMachine(
	name string,
	login string,
	password string,
	account string,
) Machine {
	return newMachine(name, login, password, account)
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

// PutMachine adds the given Machine to the configured netrc file.
func PutMachine(envContainer app.EnvContainer, machine Machine) error {
	filePath, err := getFilePath(envContainer)
	if err != nil {
		return err
	}
	return putMachineForFilePath(machine, filePath)
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

func putMachineForFilePath(machine Machine, filePath string) (retErr error) {
	file, err := os.Open(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// If a netrc file does not already exist, create one and continue.
		file, err = os.Create(filePath)
		if err != nil {
			return err
		}
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	netrc, err := netrc.Parse(file)
	if err != nil {
		return err
	}
	if foundMachine := netrc.FindMachine(machine.Name()); foundMachine != nil {
		// If the machine already exists, remove it so that its entry is overwritten.
		netrc.RemoveMachine(machine.Name())
	}
	// Put the machine into the user's netrc.
	_ = netrc.NewMachine(
		machine.Name(),
		machine.Login(),
		machine.Password(),
		machine.Account(),
	)
	bytes, err := netrc.MarshalText()
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, bytes, info.Mode())
}
