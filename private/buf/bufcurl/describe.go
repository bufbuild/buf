package bufcurl

import (
	"encoding/json"
	"fmt"
	"io"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// DescribeMessage describes the given message to the given writer using
// JSON-with-comments format. The comments indicate the Protobuf type and
// cardinality of each field and also indicate the enum values for enum
// fields.
func DescribeMessage(message protoreflect.MessageDescriptor, out io.Writer) error {
	return (&describer{
		out:              out,
		alreadyDescribed: map[protoreflect.FullName]struct{}{},
	}).describeMessage(message, "", false)
}

type describer struct {
	out              io.Writer
	indentPrefix     string
	alreadyDescribed map[protoreflect.FullName]struct{}
}

func (d *describer) indent() {
	d.indentPrefix += "  "
}

func (d *describer) unindent() {
	d.indentPrefix = d.indentPrefix[:len(d.indentPrefix)-2]
}

func (d *describer) describeMessage(
	message protoreflect.MessageDescriptor,
	commentPrefix string,
	trailingComma bool,
) error {
	fields := message.Fields()
	numFields := fields.Len()
	var comment string
	if commentPrefix == "" {
		comment = " // " + string(message.FullName())
	} else {
		comment = " // " + commentPrefix + " " + string(message.FullName())
	}
	_, alreadyDescribed := d.alreadyDescribed[message.FullName()]
	if alreadyDescribed {
		comment += " (described above)"
	} else {
		d.alreadyDescribed[message.FullName()] = struct{}{}
	}

	if numFields == 0 || alreadyDescribed {
		_, err := fmt.Fprintf(d.out, "{}%s%s\n", trailer(trailingComma), comment)
		return err
	}

	if _, err := fmt.Fprintf(d.out, "{%s\n", comment); err != nil {
		return err
	}

	d.indent()
	for i := 0; i < numFields; i++ {
		if err := d.describeField(fields.Get(i), i != numFields-1); err != nil {
			return err
		}
	}
	d.unindent()

	if _, err := fmt.Fprintf(d.out, "%s}%s\n", d.indentPrefix, trailer(trailingComma)); err != nil {
		return err
	}
	return nil
}

func (d *describer) describeField(
	field protoreflect.FieldDescriptor,
	trailingComma bool,
) error {
	fieldName, err := json.Marshal(field.JSONName())
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(d.out, "%s%s: ", d.indentPrefix, fieldName); err != nil {
		return err
	}
	switch {
	case field.IsMap():
		return d.describeMap(field.MapKey(), field.MapValue(), trailingComma)
	case field.IsList():
		return d.describeList(field, trailingComma)
	case field.Message() != nil:
		return d.describeMessage(field.Message(), field.Cardinality().String(), trailingComma)
	default:
		return d.describeScalar(field, false, trailingComma)
	}
}

func (d *describer) describeMap(
	keyField protoreflect.FieldDescriptor,
	valueField protoreflect.FieldDescriptor,
	trailingComma bool,
) error {
	var valueType string
	switch {
	case valueField.Enum() != nil:
		valueType = string(valueField.Enum().FullName())
	case valueField.Message() != nil:
		valueType = string(valueField.Message().FullName())
	default:
		valueType = valueField.Kind().String()
	}
	if _, err := fmt.Fprintf(d.out, "{ // map<%s, %s>\n", keyField.Kind().String(), valueType); err != nil {
		return err
	}

	d.indent()
	key, err := scalarValue(keyField)
	if err != nil {
		return err
	}
	if key[0] != '"' {
		// map keys must be strings in JSON, so wrap in quotes
		key = "\"" + key + "\""
	}
	if _, err := fmt.Fprintf(d.out, "%s%s: ", d.indentPrefix, key); err != nil {
		return err
	}
	if valueField.Message() != nil {
		err = d.describeMessage(valueField.Message(), "", false)
	} else {
		err = d.describeScalar(valueField, false, false)
	}
	if err != nil {
		return err
	}
	d.unindent()

	if _, err := fmt.Fprintf(d.out, "%s}%s\n", d.indentPrefix, trailer(trailingComma)); err != nil {
		return err
	}
	return nil
}

func (d *describer) describeList(
	field protoreflect.FieldDescriptor,
	trailingComma bool,
) error {
	if field.Message() == nil {
		// If not a message, must be a scalar value.
		return d.describeScalar(field, true, trailingComma)
	}
	d.indent()
	if _, err := fmt.Fprintf(d.out, "[\n%s", d.indentPrefix); err != nil {
		return err
	}
	if err := d.describeMessage(field.Message(), field.Cardinality().String(), false); err != nil {
		return err
	}
	d.unindent()
	_, err := fmt.Fprintf(d.out, "%s]%s\n", d.indentPrefix, trailer(trailingComma))
	return err
}

func (d *describer) describeScalar(
	field protoreflect.FieldDescriptor,
	inArray bool,
	trailingComma bool,
) error {
	if inArray {
		if _, err := fmt.Fprint(d.out, "[ "); err != nil {
			return err
		}
	}
	if err := d.describeScalarValue(field); err != nil {
		return err
	}
	if inArray {
		if _, err := fmt.Fprint(d.out, " ]"); err != nil {
			return err
		}
	}

	var comment string
	if !field.ContainingMessage().IsMapEntry() { // no need to print this for map values
		comment = " // " + cardinalityComment(field)
	}
	if _, err := fmt.Fprintf(d.out, "%s%s\n", trailer(trailingComma), comment); err != nil {
		return err
	}
	if field.Enum() != nil {
		if err := d.describeEnumValues(field.Enum()); err != nil {
			return err
		}
	}
	return nil
}

func (d *describer) describeScalarValue(field protoreflect.FieldDescriptor) error {
	str, err := scalarValue(field)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(d.out, str)
	return err
}

func (d *describer) describeEnumValues(enum protoreflect.EnumDescriptor) error {
	_, alreadyDescribed := d.alreadyDescribed[enum.FullName()]
	if alreadyDescribed {
		if _, err := fmt.Fprintf(d.out, "%s    // %s values (described above)\n", d.indentPrefix, enum.FullName()); err != nil {
			return err
		}
		return nil
	}
	d.alreadyDescribed[enum.FullName()] = struct{}{}

	if _, err := fmt.Fprintf(d.out, "%s    // %s values:\n", d.indentPrefix, enum.FullName()); err != nil {
		return err
	}
	vals := enum.Values()
	for i, length := 0, vals.Len(); i < length; i++ {
		val := vals.Get(i)
		if _, err := fmt.Fprintf(d.out, "%s    //     %s (%d)\n", d.indentPrefix, val.Name(), val.Number()); err != nil {
			return err
		}
	}
	return nil
}

func trailer(trailingComma bool) string {
	if trailingComma {
		return ","
	}
	return ""
}

func cardinalityComment(field protoreflect.FieldDescriptor) string {
	var suffix string
	if field.Cardinality() == protoreflect.Optional && !field.HasPresence() {
		suffix = " (implicit presence)"
	}
	return fmt.Sprintf("%s %s%s", field.Cardinality().String(), field.Kind().String(), suffix)
}

func scalarValue(field protoreflect.FieldDescriptor) (string, error) {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return "true", nil
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return "0.0", nil
	case protoreflect.Int32Kind,
		protoreflect.Sint32Kind,
		protoreflect.Uint32Kind,
		protoreflect.Fixed32Kind,
		protoreflect.Sfixed32Kind:
		return "0", nil
	case protoreflect.Int64Kind,
		protoreflect.Sint64Kind,
		protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind,
		protoreflect.Sfixed64Kind:
		return `"0"`, nil // 64 bit ints are formatted as strings to preserve precision
	case protoreflect.StringKind, protoreflect.BytesKind:
		return `""`, nil
	case protoreflect.EnumKind:
		return fmt.Sprintf(`"%s"`, field.Enum().Values().Get(0).Name()), nil
	default:
		return "", fmt.Errorf("kind %s is not scalar", field.Kind())
	}
}
