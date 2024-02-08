// Copyright 2020-2024 Buf Technologies, Inc.
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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/jdx/go-netrc"
)

// Filename exposes the netrc filename based on the current operating system.
const Filename = netrcFilename

// Machine is a machine.
type Machine interface {
	// Empty for default machine.
	Name() string
	Login() string
	Password() string
}

// NewMachine creates a new Machine.
func NewMachine(
	name string,
	login string,
	password string,
) Machine {
	return newMachine(name, login, password)
}

// GetMachineForName returns the Machine for the given name.
//
// Returns nil if no such Machine.
func GetMachineForName(envContainer app.EnvContainer, name string) (_ Machine, retErr error) {
	filePath, err := GetFilePath(envContainer)
	if err != nil {
		return nil, err
	}
	return GetMachineForNameAndFilePath(name, filePath)
}

// PutMachines adds the given Machines to the configured netrc file.
func PutMachines(envContainer app.EnvContainer, machines ...Machine) error {
	filePath, err := GetFilePath(envContainer)
	if err != nil {
		return err
	}
	return putMachinesForFilePath(machines, filePath)
}

// DeleteMachineForName deletes the Machine for the given name, if set.
//
// Returns false if there was no Machine for the given name.
func DeleteMachineForName(envContainer app.EnvContainer, name string) (bool, error) {
	filePath, err := GetFilePath(envContainer)
	if err != nil {
		return false, err
	}
	return deleteMachineForFilePath(name, filePath)
}

// GetFilePath gets the netrc file path for the given environment.
func GetFilePath(envContainer app.EnvContainer) (string, error) {
	if netrcFilePath := envContainer.Env("NETRC"); netrcFilePath != "" {
		return netrcFilePath, nil
	}
	homeDirPath, err := app.HomeDirPath(envContainer)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDirPath, netrcFilename), nil
}

// GetMachineForNameAndFilePath returns the Machine for the given name from the
// file at the given path.
//
// Returns nil if no such Machine or no such file.
func GetMachineForNameAndFilePath(name string, filePath string) (_ Machine, retErr error) {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	netrcStruct, err := netrc.Parse(filePath)
	if err != nil {
		return nil, err
	}
	netrcMachine := netrcStruct.Machine(name)
	if netrcMachine == nil {
		netrcMachine = netrcStruct.Machine("default")
		if netrcMachine == nil {
			return nil, nil
		}
	}
	// We take the name from the read Machine just in case there's some case-insensitivity weirdness
	machineName := netrcMachine.Name
	if machineName == "default" {
		machineName = ""
	}
	return newMachine(
		machineName,
		netrcMachine.Get("login"),
		netrcMachine.Get("password"),
	), nil
}

func putMachinesForFilePath(machines []Machine, filePath string) (retErr error) {
	var netrcStruct *netrc.Netrc
	fileInfo, err := os.Stat(filePath)
	var fileMode fs.FileMode
	if err != nil {
		if os.IsNotExist(err) {
			netrcStruct = &netrc.Netrc{}
			fileMode = 0600
		} else {
			return err
		}
	} else {
		netrcStruct, err = netrc.Parse(filePath)
		if err != nil {
			return err
		}
		fileMode = fileInfo.Mode()
	}
	for _, machine := range machines {
		if foundMachine := netrcStruct.Machine(machine.Name()); foundMachine != nil {
			netrcStruct.RemoveMachine(machine.Name())
		}
		netrcStruct.AddMachine(
			machine.Name(),
			machine.Login(),
			machine.Password(),
		)
	}
	return os.WriteFile(filePath, []byte(netrcStruct.Render()), fileMode)
}

func deleteMachineForFilePath(name string, filePath string) (_ bool, retErr error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If a netrc file does not already exist, there's nothing to be done.
			return false, nil
		}
		return false, err
	}
	netrcStruct, err := netrc.Parse(filePath)
	if err != nil {
		return false, err
	}
	if netrcStruct.Machine(name) == nil {
		// Machine is not set, there is nothing to be done.
		return false, nil
	}
	netrcStruct.RemoveMachine(name)
	if err := os.WriteFile(filePath, []byte(netrcStruct.Render()), fileInfo.Mode()); err != nil {
		return false, err
	}
	return true, nil
}
