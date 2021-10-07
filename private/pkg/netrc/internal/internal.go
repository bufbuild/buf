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

// Package internal is a direct copy of github.com/bgentry/go-netrc with bug fixes.
package internal

// Our bug fixes are surrounded by *** BUGFIX *** below.
//
// Except for our edits, this code is
// Copyright © 2010 Fazlul Shahriar <fshahriar@gmail.com> and
// Copyright © 2014 Blake Gentry <blakesgentry@gmail.com>.
//
// See https://github.com/bgentry/go-netrc/blob/9fd32a8b3d3d3f9d43c341bfe098430e07609480/LICENSE
// for the original license.

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type tkType int

const (
	tkMachine tkType = iota
	tkDefault
	tkLogin
	tkPassword
	tkAccount
	tkMacdef
	tkComment
	tkWhitespace
)

var keywords = map[string]tkType{
	"machine":  tkMachine,
	"default":  tkDefault,
	"login":    tkLogin,
	"password": tkPassword,
	"account":  tkAccount,
	"macdef":   tkMacdef,
	"#":        tkComment,
}

type Netrc struct {
	tokens     []*token
	machines   []*Machine
	macros     Macros
	updateLock sync.Mutex
}

// FindMachine returns the Machine in n named by name. If a machine named by
// name exists, it is returned. If no Machine with name name is found and there
// is a ``default'' machine, the ``default'' machine is returned. Otherwise, nil
// is returned.
func (n *Netrc) FindMachine(name string) (m *Machine) {
	// TODO(bgentry): not safe for concurrency
	var def *Machine
	for _, m = range n.machines {
		if m.Name == name {
			return m
		}
		if m.IsDefault() {
			def = m
		}
	}
	if def == nil {
		return nil
	}
	return def
}

// MarshalText implements the encoding.TextMarshaler interface to encode a
// Netrc into text format.
func (n *Netrc) MarshalText() (text []byte, err error) {
	// TODO(bgentry): not safe for concurrency
	for i := range n.tokens {
		switch n.tokens[i].kind {
		case tkComment, tkDefault, tkWhitespace: // always append these types
			text = append(text, n.tokens[i].rawkind...)
		default:
			if n.tokens[i].value != "" { // skip empty-value tokens
				text = append(text, n.tokens[i].rawkind...)
			}
		}
		if n.tokens[i].kind == tkMacdef {
			text = append(text, ' ')
			text = append(text, n.tokens[i].macroName...)
		}
		text = append(text, n.tokens[i].rawvalue...)
	}
	return
}

func (n *Netrc) NewMachine(name, login, password, account string) *Machine {
	n.updateLock.Lock()
	defer n.updateLock.Unlock()

	m := &Machine{
		Name:     name,
		Login:    login,
		Password: password,
		Account:  account,

		// *** BUGFIX ***
		// https://github.com/bufbuild/buf/issues/642
		nametoken:    n.newNameToken(name),
		logintoken:   newLoginToken(login),
		passtoken:    newPassToken(password),
		accounttoken: newAccountToken(account),
		// *** BUGFIX END ***
	}
	n.insertMachineTokensBeforeDefault(m)
	for i := range n.machines {
		if n.machines[i].IsDefault() {
			n.machines = append(append(n.machines[:i], m), n.machines[i:]...)
			return m
		}
	}
	n.machines = append(n.machines, m)
	return m
}

func (n *Netrc) newNameToken(value string) *token {
	prefix := "\n"
	if len(n.tokens) == 0 {
		prefix = ""
	}
	var rawvalue []byte
	if value != "" {
		rawvalue = []byte(" " + value)
	}
	return &token{
		kind:     tkMachine,
		rawkind:  []byte(prefix + "machine"),
		value:    value,
		rawvalue: rawvalue,
	}
}

func newLoginToken(value string) *token {
	var rawvalue []byte
	if value != "" {
		rawvalue = []byte(" " + value)
	}
	return &token{
		kind:     tkLogin,
		rawkind:  []byte("\n\tlogin"),
		value:    value,
		rawvalue: rawvalue,
	}
}

func newPassToken(value string) *token {
	var rawvalue []byte
	if value != "" {
		rawvalue = []byte(" " + value)
	}
	return &token{
		kind:     tkPassword,
		rawkind:  []byte("\n\tpassword"),
		value:    value,
		rawvalue: rawvalue,
	}
}

func newAccountToken(value string) *token {
	var rawvalue []byte
	if value != "" {
		rawvalue = []byte(" " + value)
	}
	return &token{
		kind:     tkAccount,
		rawkind:  []byte("\n\taccount"),
		value:    value,
		rawvalue: rawvalue,
	}
}

func (n *Netrc) insertMachineTokensBeforeDefault(m *Machine) {
	newtokens := []*token{m.nametoken}
	//if m.logintoken.value != "" {
	newtokens = append(newtokens, m.logintoken)
	//}
	//if m.passtoken.value != "" {
	newtokens = append(newtokens, m.passtoken)
	//}
	//if m.accounttoken.value != "" {
	newtokens = append(newtokens, m.accounttoken)
	//}
	for i := range n.tokens {
		if n.tokens[i].kind == tkDefault {
			// found the default, now insert tokens before it
			n.tokens = append(n.tokens[:i], append(newtokens, n.tokens[i:]...)...)
			return
		}
	}
	// didn't find a default, just add the newtokens to the end
	n.tokens = append(n.tokens, newtokens...)
}

func (n *Netrc) RemoveMachine(name string) {
	n.updateLock.Lock()
	defer n.updateLock.Unlock()

	for i := range n.machines {
		if n.machines[i] != nil && n.machines[i].Name == name {
			m := n.machines[i]
			for _, t := range []*token{
				m.nametoken, m.logintoken, m.passtoken, m.accounttoken,
			} {
				n.removeToken(t)
			}
			n.machines = append(n.machines[:i], n.machines[i+1:]...)
			return
		}
	}
}

func (n *Netrc) removeToken(t *token) {
	if t != nil {
		for i := range n.tokens {
			if n.tokens[i] == t {
				n.tokens = append(n.tokens[:i], n.tokens[i+1:]...)
				return
			}
		}
	}
}

// Machine contains information about a remote machine.
type Machine struct {
	Name     string
	Login    string
	Password string
	Account  string

	nametoken    *token
	logintoken   *token
	passtoken    *token
	accounttoken *token
}

// IsDefault returns true if the machine is a "default" token, denoted by an
// empty name.
func (m *Machine) IsDefault() bool {
	return m.Name == ""
}

// UpdatePassword sets the password for the Machine m.
func (m *Machine) UpdatePassword(newpass string) {
	m.Password = newpass
	// *** BUGFIX ***
	// https://github.com/bufbuild/buf/issues/642
	if newpass == "" {
		m.passtoken = nil
		return
	}
	if m.passtoken == nil {
		m.passtoken = newPassToken("")
	}
	// *** BUGFIX END ***
	updateTokenValue(m.passtoken, newpass)
}

// UpdateLogin sets the login for the Machine m.
func (m *Machine) UpdateLogin(newlogin string) {
	m.Login = newlogin
	// *** BUGFIX ***
	// https://github.com/bufbuild/buf/issues/642
	if newlogin == "" {
		m.logintoken = nil
		return
	}
	if m.logintoken == nil {
		m.logintoken = newLoginToken("")
	}
	// *** BUGFIX END ***
	updateTokenValue(m.logintoken, newlogin)
}

// UpdateAccount sets the login for the Machine m.
func (m *Machine) UpdateAccount(newaccount string) {
	m.Account = newaccount
	// *** BUGFIX ***
	// https://github.com/bufbuild/buf/issues/642
	if newaccount == "" {
		m.accounttoken = nil
		return
	}
	if m.accounttoken == nil {
		m.accounttoken = newAccountToken("")
	}
	// *** BUGFIX END ***
	updateTokenValue(m.accounttoken, newaccount)
}

func updateTokenValue(t *token, value string) {
	oldvalue := t.value
	t.value = value
	newraw := make([]byte, len(t.rawvalue))
	copy(newraw, t.rawvalue)
	base := newraw
	// *** BUGFIX ***
	// https://github.com/bufbuild/buf/issues/642
	if len(oldvalue) > 0 {
		base = bytes.TrimSuffix(newraw, []byte(oldvalue))
	}
	t.rawvalue = append(
		base,
		[]byte(value)...,
	)
	// *** BUGFIX END ***
}

// Macros contains all the macro definitions in a netrc file.
type Macros map[string]string

type token struct {
	kind      tkType
	macroName string
	value     string
	rawkind   []byte
	rawvalue  []byte
}

// Error represents a netrc file parse error.
type Error struct {
	LineNum int    // Line number
	Msg     string // Error message
}

// Error returns a string representation of error e.
func (e *Error) Error() string {
	return fmt.Sprintf("line %d: %s", e.LineNum, e.Msg)
}

func (e *Error) BadDefaultOrder() bool {
	return e.Msg == errBadDefaultOrder
}

const errBadDefaultOrder = "default token must appear after all machine tokens"

// scanLinesKeepPrefix is a split function for a Scanner that returns each line
// of text. The returned token may include newlines if they are before the
// first non-space character. The returned line may be empty. The end-of-line
// marker is one optional carriage return followed by one mandatory newline. In
// regular expression notation, it is `\r?\n`. The last non-empty line of
// input will be returned even if it has no newline.
func scanLinesKeepPrefix(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	// Skip leading spaces.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !unicode.IsSpace(r) {
			break
		}
	}
	if i := bytes.IndexByte(data[start:], '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return start + i, data[0 : start+i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// scanWordsKeepPrefix is a split function for a Scanner that returns each
// space-separated word of text, with prefixing spaces included. It will never
// return an empty string. The definition of space is set by unicode.IsSpace.
//
// Adapted from bufio.ScanWords().
func scanTokensKeepPrefix(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading spaces.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !unicode.IsSpace(r) {
			break
		}
	}
	if atEOF && len(data) == 0 || start == len(data) {
		return len(data), data, nil
	}
	if len(data) > start && data[start] == '#' {
		return scanLinesKeepPrefix(data, atEOF)
	}
	// Scan until space, marking end of word.
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if unicode.IsSpace(r) {
			return i, data[:i], nil
		}
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func newToken(rawb []byte) (*token, error) {
	_, tkind, err := bufio.ScanWords(rawb, true)
	if err != nil {
		return nil, err
	}
	var ok bool
	t := token{rawkind: rawb}
	t.kind, ok = keywords[string(tkind)]
	if !ok {
		trimmed := strings.TrimSpace(string(tkind))
		if trimmed == "" {
			t.kind = tkWhitespace // whitespace-only, should happen only at EOF
			return &t, nil
		}
		if strings.HasPrefix(trimmed, "#") {
			t.kind = tkComment // this is a comment
			return &t, nil
		}
		return &t, fmt.Errorf("keyword expected; got " + string(tkind))
	}
	return &t, nil
}

func scanValue(scanner *bufio.Scanner, pos int) ([]byte, string, int, error) {
	if scanner.Scan() {
		raw := scanner.Bytes()
		pos += bytes.Count(raw, []byte{'\n'})
		return raw, strings.TrimSpace(string(raw)), pos, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, "", pos, &Error{pos, err.Error()}
	}
	return nil, "", pos, nil
}

func parse(r io.Reader, pos int) (*Netrc, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	nrc := Netrc{machines: make([]*Machine, 0, 20), macros: make(Macros, 10)}

	defaultSeen := false
	var currentMacro *token
	var m *Machine
	var t *token
	scanner := bufio.NewScanner(bytes.NewReader(b))
	// *** BUGFIX ***
	// https://github.com/bufbuild/buf/issues/611
	// Without this, parsing fails for files larger than 4096 bytes
	if len(b) > 4096 {
		buffer := make([]byte, len(b))
		maxTokenSize := 1 << 20
		if len(b) > maxTokenSize {
			maxTokenSize = len(b) * 8
		}
		scanner.Buffer(buffer, maxTokenSize)
	}
	// *** BUGFIX END ***
	scanner.Split(scanTokensKeepPrefix)

	for scanner.Scan() {
		rawb := scanner.Bytes()
		if len(rawb) == 0 {
			break
		}
		pos += bytes.Count(rawb, []byte{'\n'})
		t, err = newToken(rawb)
		if err != nil {
			if currentMacro == nil {
				return nil, &Error{pos, err.Error()}
			}
			currentMacro.rawvalue = append(currentMacro.rawvalue, rawb...)
			continue
		}

		if currentMacro != nil && bytes.Contains(rawb, []byte{'\n', '\n'}) {
			// if macro rawvalue + rawb would contain \n\n, then macro def is over
			currentMacro.value = strings.TrimLeft(string(currentMacro.rawvalue), "\r\n")
			nrc.macros[currentMacro.macroName] = currentMacro.value
			currentMacro = nil
		}

		switch t.kind {
		case tkMacdef:
			if _, t.macroName, pos, err = scanValue(scanner, pos); err != nil {
				return nil, &Error{pos, err.Error()}
			}
			currentMacro = t
		case tkDefault:
			if defaultSeen {
				return nil, &Error{pos, "multiple default token"}
			}
			if m != nil {
				nrc.machines, m = append(nrc.machines, m), nil
			}
			m = new(Machine)
			m.Name = ""
			defaultSeen = true
		case tkMachine:
			if defaultSeen {
				return nil, &Error{pos, errBadDefaultOrder}
			}
			if m != nil {
				nrc.machines, m = append(nrc.machines, m), nil
			}
			m = new(Machine)
			if t.rawvalue, m.Name, pos, err = scanValue(scanner, pos); err != nil {
				return nil, &Error{pos, err.Error()}
			}
			t.value = m.Name
			m.nametoken = t
		case tkLogin:
			if m == nil || m.Login != "" {
				return nil, &Error{pos, "unexpected token login "}
			}
			if t.rawvalue, m.Login, pos, err = scanValue(scanner, pos); err != nil {
				return nil, &Error{pos, err.Error()}
			}
			t.value = m.Login
			m.logintoken = t
		case tkPassword:
			if m == nil || m.Password != "" {
				return nil, &Error{pos, "unexpected token password"}
			}
			if t.rawvalue, m.Password, pos, err = scanValue(scanner, pos); err != nil {
				return nil, &Error{pos, err.Error()}
			}
			t.value = m.Password
			m.passtoken = t
		case tkAccount:
			if m == nil || m.Account != "" {
				return nil, &Error{pos, "unexpected token account"}
			}
			if t.rawvalue, m.Account, pos, err = scanValue(scanner, pos); err != nil {
				return nil, &Error{pos, err.Error()}
			}
			t.value = m.Account
			m.accounttoken = t
		}

		nrc.tokens = append(nrc.tokens, t)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if m != nil {
		nrc.machines, m = append(nrc.machines, m), nil
	}
	return &nrc, nil
}

// ParseFile opens the file at filename and then passes its io.Reader to
// Parse().
func ParseFile(filename string) (*Netrc, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return Parse(fd)
}

// Parse parses from the the Reader r as a netrc file and returns the set of
// machine information and macros defined in it. The ``default'' machine,
// which is intended to be used when no machine name matches, is identified
// by an empty machine name. There can be only one ``default'' machine.
//
// If there is a parsing error, an Error is returned.
func Parse(r io.Reader) (*Netrc, error) {
	return parse(r, 1)
}

// FindMachine parses the netrc file identified by filename and returns the
// Machine named by name. If a problem occurs parsing the file at filename, an
// error is returned. If a machine named by name exists, it is returned. If no
// Machine with name name is found and there is a ``default'' machine, the
// ``default'' machine is returned. Otherwise, nil is returned.
func FindMachine(filename, name string) (m *Machine, err error) {
	n, err := ParseFile(filename)
	if err != nil {
		return nil, err
	}
	return n.FindMachine(name), nil
}
