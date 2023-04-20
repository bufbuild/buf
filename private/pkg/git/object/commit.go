// Copyright 2020-2023 Buf Technologies, Inc.
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

package object

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Ident is a git user identifier. You'll find these in author, commiter, and
// other values for user identification.
type Ident struct {
	Name      string
	Email     string
	Timestamp time.Time
}

func (i *Ident) UnmarshalText(data []byte) error {
	// Name (optional)
	// Many spaces between name and email are allowed.
	name, emailAndTime, found := strings.Cut(string(data), "<")
	if !found {
		// Mail is required.
		return errors.New("ident: no email component")
	}
	i.Name = strings.TrimRight(name, " ")

	// Email (required)
	idx := strings.LastIndex(emailAndTime, ">")
	if idx == -1 {
		return errors.New("ident: malformed email component")
	}
	i.Email = emailAndTime[:idx]

	// Timestamp (optional)
	// The stamp is in Unix Epoc and the user's UTC offset in [+-]HHMM when the
	// time was taken.
	timestr := strings.TrimLeft(emailAndTime[idx+1:], " ")
	if timestr == "" {
		return nil
	}
	timesecstr, timezonestr, found := strings.Cut(timestr, " ")
	if !found {
		return errors.New("ident: malformed timestamp: missing UTC offset")
	}
	timesec, err := strconv.ParseInt(timesecstr, 10, 64)
	if err != nil {
		return fmt.Errorf("ident: malformed timestamp: %w", err)
	}
	tzHourStr := timezonestr[:len(timezonestr)-2]
	tzHour, err := strconv.ParseInt(tzHourStr, 10, 32)
	if err != nil {
		return fmt.Errorf("ident: malformed timestamp: %w", err)
	}
	tzMinStr := timezonestr[len(timezonestr)-2:]
	tzMin, err := strconv.ParseInt(tzMinStr, 10, 32)
	if err != nil {
		return fmt.Errorf("ident: malformed timestamp: %w", err)
	}
	tzOffset := int(tzHour)*60*60 + int(tzMin)*60
	location := time.FixedZone("UTC"+timezonestr, tzOffset)
	i.Timestamp = time.Unix(timesec, 0).In(location)
	return nil
}

// Commit represents a commit object. A valid object will have a Tree.
type Commit struct {
	Tree      ID
	Parents   []ID
	Author    Ident
	Committer Ident
	Message   string
}

func (c *Commit) unmarshalHeader(header, value string) error {
	switch header {
	case "tree":
		if c.Tree != nil {
			return errors.New("too many tree headers")
		}
		if err := c.Tree.UnmarshalText([]byte(value)); err != nil {
			return err
		}
	case "parent":
		var hash ID
		if err := hash.UnmarshalText([]byte(value)); err != nil {
			return err
		}
		c.Parents = append(c.Parents, hash)
	case "author":
		if err := c.Author.UnmarshalText([]byte(value)); err != nil {
			return err
		}
	case "committer":
		if err := c.Committer.UnmarshalText([]byte(value)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Commit) UnmarshalText(data []byte) error {
	c.Tree = nil
	c.Parents = nil
	buffer := bytes.NewBuffer(data)
	// Headers
	line, err := buffer.ReadString('\n')
	for err != io.EOF && line != "\n" {
		header, value, _ := strings.Cut(line, " ")
		value = strings.TrimRight(value, "\n")
		if err := c.unmarshalHeader(header, value); err != nil {
			return err
		}
		line, err = buffer.ReadString('\n')
	}
	// Message
	c.Message = buffer.String()
	return nil
}
