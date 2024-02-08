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

package git

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ident struct {
	name      string
	email     string
	timestamp time.Time
}

func (i *ident) Name() string {
	return i.name
}
func (i *ident) Email() string {
	return i.email
}
func (i *ident) Timestamp() time.Time {
	return i.timestamp
}

func parseIdent(data []byte) (*ident, error) {
	var i ident
	// Name (optional)
	// Many spaces between name and email are allowed.
	name, emailAndTime, found := strings.Cut(string(data), "<")
	if !found {
		// Mail is required.
		return nil, errors.New("ident: no email component")
	}
	i.name = strings.TrimRight(name, " ")

	// Email (required)
	idx := strings.LastIndex(emailAndTime, ">")
	if idx == -1 {
		return nil, errors.New("ident: malformed email component")
	}
	i.email = emailAndTime[:idx]

	// Timestamp (optional)
	// The stamp is in Unix Epoc and the user's UTC offset in [+-]HHMM when the
	// time was taken.
	timestr := strings.TrimLeft(emailAndTime[idx+1:], " ")
	if timestr != "" {
		timesecstr, timezonestr, found := strings.Cut(timestr, " ")
		if !found {
			return nil, errors.New("ident: malformed timestamp: missing UTC offset")
		}
		timesec, err := strconv.ParseInt(timesecstr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("ident: malformed timestamp: %w", err)
		}
		tzHourStr := timezonestr[:len(timezonestr)-2]
		tzHour, err := strconv.ParseInt(tzHourStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("ident: malformed timestamp: %w", err)
		}
		tzMinStr := timezonestr[len(timezonestr)-2:]
		tzMin, err := strconv.ParseInt(tzMinStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("ident: malformed timestamp: %w", err)
		}
		tzOffset := int(tzHour)*60*60 + int(tzMin)*60
		location := time.FixedZone("UTC"+timezonestr, tzOffset)
		i.timestamp = time.Unix(timesec, 0).In(location)
	}
	return &i, nil
}
