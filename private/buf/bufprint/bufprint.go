// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufprint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/standard/xstrings"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/protostat"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// FormatText is the text format.
	FormatText Format = 1
	// FormatJSON is the JSON format.
	FormatJSON Format = 2
)

var (
	// AllFormatsString is the string representation of all Formats.
	AllFormatsString = xstrings.SliceToString([]string{FormatText.String(), FormatJSON.String()})
)

// Format is a format to print.
type Format int

// ParseFormat parses the format.
//
// If the empty string is provided, this is interpreted as FormatText.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "", "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return 0, fmt.Errorf("unknown format: %s", s)
	}
}

// String implements fmt.Stringer.
func (f Format) String() string {
	switch f {
	case FormatText:
		return "text"
	case FormatJSON:
		return "json"
	default:
		return strconv.Itoa(int(f))
	}
}

// Entity is an entity printed structurally by functions in bufprint package.
// It's used in "buf registry" commands where the CLI prints a page of entities, such as
// commits, an entity's info or simply an entity's full name.
//
// When printed by PrintEntity in text format, any field with a field tag in the form
// of `bufprint:"<field name>[,omitempty]"` is printed.
//
// This means that an implementation of Entity must also be a struct.
type Entity interface {
	fullName() string
}

// PrintNames prints entities' names.
//
// If format is FormatJSON, this also prints information about each entity, the
// same as calling PrintInfo on each entity.
func PrintNames(writer io.Writer, format Format, entities ...Entity) error {
	switch format {
	case FormatJSON:
		for _, entity := range entities {
			if err := json.NewEncoder(writer).Encode(entity); err != nil {
				return err
			}
		}
		return nil
	case FormatText:
		for _, entity := range entities {
			if _, err := fmt.Fprintln(writer, entity.fullName()); err != nil {
				return err
			}
		}
		return nil
	default:
		return syserror.Newf("unknown format: %s", format)
	}
}

// PrintPage prints a page of entities.
func PrintPage(
	writer io.Writer,
	format Format,
	nextPageToken string,
	nextPageCommand string,
	entities []Entity,
) error {
	if len(entities) == 0 {
		return nil
	}
	var entitiesName string
	for _, entity := range entities {
		var currentEntitiesName string
		switch entity.(type) {
		case outputLabel:
			currentEntitiesName = "labels"
		case outputCommit:
			currentEntitiesName = "commits"
		case outputModule:
			currentEntitiesName = "modules"
		case outputOrganization:
			currentEntitiesName = "organizations"
		default:
			return syserror.Newf("unknown implementation of Entity: %T", entity)
		}
		if currentEntitiesName != entitiesName && entitiesName != "" {
			return syserror.Newf("the page has both %s and %s", currentEntitiesName, entitiesName)
		}
		entitiesName = currentEntitiesName
	}
	switch format {
	case FormatText:
		if err := PrintNames(writer, format, entities...); err != nil {
			return err
		}
		if nextPageToken == "" {
			return nil
		}
		_, err := fmt.Fprintf(
			writer,
			"\nMore than %d %s found, run %q to list more\n",
			len(entities),
			entitiesName,
			nextPageCommand,
		)
		return err
	case FormatJSON:
		return json.NewEncoder(writer).Encode(&entityPage{
			NextPage:         nextPageToken,
			Entities:         entities,
			pluralEntityName: entitiesName,
		})
	default:
		return syserror.Newf("unknown format: %v", format)
	}
}

// PrintEntity prints an entity.
//
// If format is FormatText, this prints the information in a table.
// If format is FormatJSON, this prints the information as a JSON object.
func PrintEntity(writer io.Writer, format Format, entity Entity) error {
	switch format {
	case FormatJSON:
		return json.NewEncoder(writer).Encode(entity)
	case FormatText:
		fieldNames, fieldValues, err := getFieldNamesAndValuesForInfo(entity)
		if err != nil {
			return err
		}
		return WithTabWriter(
			writer,
			fieldNames,
			func(tabWriter TabWriter) error {
				return tabWriter.Write(fieldValues...)
			},
		)
	default:
		return syserror.Newf("unknown format: %s", format)
	}
}

// NewLabelEntity returns a new label entity to print. It takes a label as an
// interface to allow for modulev1.Label and pluginv1beta1.Label to be passed.
func NewLabelEntity(label interface {
	GetName() string
	GetCommitId() string
	GetCreateTime() *timestamppb.Timestamp
	GetArchiveTime() *timestamppb.Timestamp
}, moduleFullName bufparse.FullName) Entity {
	var archiveTime *time.Time
	if label.GetArchiveTime() != nil {
		timeValue := label.GetArchiveTime().AsTime()
		archiveTime = &timeValue
	}
	return outputLabel{
		Name:           label.GetName(),
		Commit:         label.GetCommitId(),
		CreateTime:     label.GetCreateTime().AsTime(),
		ArchiveTime:    archiveTime,
		entityFullName: moduleFullName,
	}
}

// NewCommitEntity returns a new commit entity to print. It takes a commit as
// an interface to allow for modulev1.Commit and pluginv1beta1.Commit to be passed.
func NewCommitEntity(
	commit interface {
		GetId() string
		GetCreateTime() *timestamppb.Timestamp
	},
	moduleFullName bufparse.FullName,
	sourceControlURL string,
) Entity {
	return outputCommit{
		Commit:           commit.GetId(),
		CreateTime:       commit.GetCreateTime().AsTime(),
		SourceControlURL: sourceControlURL,
		entityFullName:   moduleFullName,
	}
}

// NewModuleEntity returns a new module entity to print.
func NewModuleEntity(module *modulev1.Module, moduleFullName bufparse.FullName) Entity {
	return outputModule{
		ID:               module.Id,
		Remote:           moduleFullName.Registry(),
		Owner:            moduleFullName.Owner(),
		Name:             moduleFullName.Name(),
		FullName:         moduleFullName.String(),
		CreateTime:       module.CreateTime.AsTime(),
		State:            module.State.String(),
		DefaultLabelName: module.GetDefaultLabelName(),
	}
}

// NewOrganizationEntity returns a new organization entity to print.
func NewOrganizationEntity(organization *ownerv1.Organization, remote string) Entity {
	return outputOrganization{
		ID:         organization.Id,
		Remote:     remote,
		Name:       organization.Name,
		FullName:   fmt.Sprintf("%s/%s", remote, organization.Name),
		CreateTime: organization.CreateTime.AsTime(),
	}
}

// NewPluginEntity returns a new plugin entity to print.
func NewPluginEntity(plugin *pluginv1beta1.Plugin, pluginFullName bufparse.FullName) Entity {
	return outputPlugin{
		ID:         plugin.Id,
		Remote:     pluginFullName.Registry(),
		Owner:      pluginFullName.Owner(),
		Name:       pluginFullName.Name(),
		FullName:   pluginFullName.String(),
		CreateTime: plugin.CreateTime.AsTime(),
	}
}

// NewPolicyEntity returns a new plugin entity to print.
func NewPolicyEntity(policy *policyv1beta1.Policy, policyFullName bufparse.FullName) Entity {
	return outputPolicy{
		ID:         policy.Id,
		Remote:     policyFullName.Registry(),
		Owner:      policyFullName.Owner(),
		Name:       policyFullName.Name(),
		FullName:   policyFullName.String(),
		CreateTime: policy.CreateTime.AsTime(),
	}
}

// NewUserEntity returns a new user entity to print.
func NewUserEntity(user *registryv1alpha1.User) Entity {
	return outputUser{
		Username: user.GetUsername(),
		// We use the Username as the full name for the user when printing.
		FullName: user.GetUsername(),
	}
}

// CuratedPluginPrinter is a printer for curated plugins.
type CuratedPluginPrinter interface {
	PrintCuratedPlugin(ctx context.Context, format Format, plugin *registryv1alpha1.CuratedPlugin) error
	PrintCuratedPlugins(ctx context.Context, format Format, nextPageToken string, plugins ...*registryv1alpha1.CuratedPlugin) error
}

// NewCuratedPluginPrinter returns a new CuratedPluginPrinter.
func NewCuratedPluginPrinter(writer io.Writer) CuratedPluginPrinter {
	return newCuratedPluginPrinter(writer)
}

// TokenPrinter is a token printer.
//
// TODO: update to same format as other printers.
type TokenPrinter interface {
	PrintTokens(ctx context.Context, tokens ...*registryv1alpha1.Token) error
}

// NewTokenPrinter returns a new TokenPrinter.
//
// TODO: update to same format as other printers.
func NewTokenPrinter(writer io.Writer, format Format) (TokenPrinter, error) {
	switch format {
	case FormatText:
		return newTokenTextPrinter(writer), nil
	case FormatJSON:
		return newTokenJSONPrinter(writer), nil
	default:
		return nil, fmt.Errorf("unknown format: %v", format)
	}
}

// StatsPrinter is a printer of Stats.
type StatsPrinter interface {
	PrintStats(ctx context.Context, format Format, stats *protostat.Stats) error
}

// NewStatsPrinter returns a new StatsPrinter.
func NewStatsPrinter(writer io.Writer) StatsPrinter {
	return newStatsPrinter(writer)
}

// SDKInfoPrinter is a printer for SDK info.
type SDKInfoPrinter interface {
	PrintSDKInfo(ctx context.Context, format Format, sdkInfo *registryv1alpha1.GetSDKInfoResponse) error
}

// NewSDKInfoPrinter returns a new SDKInfoPrinter.
func NewSDKInfoPrinter(writer io.Writer) SDKInfoPrinter {
	return newSDKInfoPrinter(writer)
}

// TabWriter is a tab writer.
type TabWriter interface {
	Write(values ...string) error
}

// WithTabWriter calls a function with a TabWriter.
//
// Shared with internal packages.
func WithTabWriter(
	writer io.Writer,
	header []string,
	f func(TabWriter) error,
) (retErr error) {
	tabWriter := newTabWriter(writer)
	defer func() {
		retErr = errors.Join(retErr, tabWriter.Flush())
	}()
	if err := tabWriter.Write(header...); err != nil {
		return err
	}
	return f(tabWriter)
}

// *** PRIVATE ***

// printProtoMessageJSON prints the Protobuf message as JSON.
func printProtoMessageJSON(writer io.Writer, message proto.Message) error {
	data, err := protoencoding.NewJSONMarshaler(nil, protoencoding.JSONMarshalerWithIndent()).Marshal(message)
	if err != nil {
		return err
	}
	_, err = writer.Write(append(data, []byte("\n")...))
	return err
}

func getFieldNamesAndValuesForInfo(entity any) ([]string, []string, error) {
	reflectType := reflect.TypeOf(entity)
	if reflectType.Kind() != reflect.Struct {
		return nil, nil, syserror.Newf("%T is not a struct", entity)
	}
	numField := reflectType.NumField()
	reflectValue := reflect.ValueOf(entity)
	var fieldNames []string
	var fieldValues []string
	for i := range numField {
		field := reflectType.Field(i)
		bufprintTag, ok := field.Tag.Lookup("bufprint")
		if !ok {
			continue
		}
		var fieldName string
		var omitEmpty bool
		parts := strings.SplitN(bufprintTag, ",", 2)
		switch len(parts) {
		case 1:
			fieldName = parts[0]
		case 2:
			fieldName = parts[0]
			if parts[1] != "omitempty" {
				return nil, nil, syserror.Newf("unknown bufprint tag value: %s", parts[1])
			}
			omitEmpty = true
		default:
			return nil, nil, syserror.Newf("unexpected number of parts in bufprint tag: %s", bufprintTag)
		}
		value := reflectValue.Field(i)
		switch t := value.Interface().(type) {
		case string:
			if omitEmpty && t == "" {
				continue
			}
			fieldValues = append(fieldValues, t)
		case *time.Time:
			if omitEmpty && t == nil {
				continue
			}
			var value string
			if t != nil {
				value = t.Format(time.RFC3339)
			}
			fieldValues = append(fieldValues, value)
		case time.Time:
			if omitEmpty && (t.Equal(time.Time{}) || t.Equal((&timestamppb.Timestamp{}).AsTime())) {
				continue
			}
			fieldValues = append(fieldValues, t.Format(time.RFC3339))
		default:
			return nil, nil, syserror.Newf("unexpected data type: %T", t)
		}
		fieldNames = append(fieldNames, fieldName)
	}
	return fieldNames, fieldValues, nil
}

type entityPage struct {
	NextPage string   `json:"next_page,omitempty"`
	Entities []Entity `json:"entities"`

	pluralEntityName string
}

func (p *entityPage) MarshalJSON() ([]byte, error) {
	value := reflect.ValueOf(*p)
	t := value.Type()
	fields := make([]reflect.StructField, 0)
	for i := range t.NumField() {
		fields = append(fields, t.Field(i))
		if t.Field(i).Name == "Entities" {
			fields[i].Tag = reflect.StructTag(fmt.Sprintf(`json:"%s"`, p.pluralEntityName))
		}
	}
	newType := reflect.StructOf(fields)
	newValue := value.Convert(newType)
	return json.Marshal(newValue.Interface())
}

type outputLabel struct {
	Name        string     `json:"name,omitempty" bufprint:"Name"`
	Commit      string     `json:"commit,omitempty" bufprint:"Commit"`
	CreateTime  time.Time  `json:"create_time,omitempty" bufprint:"Create Time"`
	ArchiveTime *time.Time `json:"archive_time,omitempty" bufprint:"Archive Time,omitempty"`

	entityFullName bufparse.FullName
}

func (l outputLabel) fullName() string {
	return fmt.Sprintf("%s:%s", l.entityFullName.String(), l.Name)
}

type outputCommit struct {
	Commit           string    `json:"commit,omitempty" bufprint:"Commit"`
	CreateTime       time.Time `json:"create_time,omitempty" bufprint:"Create Time"`
	SourceControlURL string    `json:"source_control_url,omitempty"`

	entityFullName bufparse.FullName
}

func (c outputCommit) fullName() string {
	return fmt.Sprintf("%s:%s", c.entityFullName.String(), c.Commit)
}

type outputModule struct {
	ID               string    `json:"id,omitempty"`
	Remote           string    `json:"remote,omitempty"`
	Owner            string    `json:"owner,omitempty"`
	Name             string    `json:"name,omitempty"`
	FullName         string    `json:"-" bufprint:"Name"`
	CreateTime       time.Time `json:"create_time,omitempty" bufprint:"Create Time"`
	State            string    `json:"state,omitempty"`
	DefaultLabelName string    `json:"default_label_name,omitempty"`
}

func (m outputModule) fullName() string {
	return m.FullName
}

type outputOrganization struct {
	ID         string    `json:"id,omitempty"`
	Remote     string    `json:"remote,omitempty"`
	Name       string    `json:"name,omitempty"`
	FullName   string    `json:"-" bufprint:"Name"`
	CreateTime time.Time `json:"create_time,omitempty" bufprint:"Create Time"`
}

func (o outputOrganization) fullName() string {
	return o.FullName
}

type outputPlugin struct {
	ID         string    `json:"id,omitempty"`
	Remote     string    `json:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty"`
	Name       string    `json:"name,omitempty"`
	FullName   string    `json:"-" bufprint:"Name"`
	CreateTime time.Time `json:"create_time,omitempty" bufprint:"Create Time"`
}

func (m outputPlugin) fullName() string {
	return m.FullName
}

type outputPolicy struct {
	ID         string    `json:"id,omitempty"`
	Remote     string    `json:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty"`
	Name       string    `json:"name,omitempty"`
	FullName   string    `json:"-" bufprint:"Name"`
	CreateTime time.Time `json:"create_time,omitempty" bufprint:"Create Time"`
}

func (m outputPolicy) fullName() string {
	return m.FullName
}

type outputUser struct {
	Username string `json:"username,omitempty"`
	FullName string `json:"-" bufprint:"Name"`
}

func (o outputUser) fullName() string {
	return o.FullName
}
