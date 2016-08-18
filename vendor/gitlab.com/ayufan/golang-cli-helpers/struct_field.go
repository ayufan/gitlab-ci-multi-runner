package clihelpers

import (
	"strings"
	"reflect"
	"github.com/codegangsta/cli"
)

type StructFieldValue struct {
	field reflect.StructField
	value reflect.Value
}

func (f StructFieldValue) IsBoolFlag() bool {
	if f.value.Kind() == reflect.Bool {
		return true
	} else if f.value.Kind() == reflect.Ptr && f.value.Elem().Kind() == reflect.Bool {
		return true
	} else {
		return false
	}
}

func (s StructFieldValue) Set(val string) error {
	return convert(val, s.value, s.field.Tag)
}

func (s StructFieldValue) String() string {
	if s.value.Kind() == reflect.Ptr && s.value.IsNil() {
		return ""
	}
	retval, _ := convertToString(s.value, s.field.Tag)
	return retval
}

type StructFieldFlag struct {
	cli.GenericFlag
}

func (f StructFieldFlag) String() string {
	if sf, ok := f.Value.(StructFieldValue); ok {
		if sf.IsBoolFlag() {
			flag := &cli.BoolFlag{
				Name: f.Name,
				Usage: f.Usage,
				EnvVar: f.EnvVar,
			}
			return flag.String()
		} else {
			flag := &cli.StringFlag{
				Name: f.Name,
				Value: sf.String(),
				Usage: f.Usage,
				EnvVar: f.EnvVar,
			}
			return flag.String()
		}
	} else {
		return f.GenericFlag.String()
	}
}

func getStructFieldFlag(field reflect.StructField, fieldValue reflect.Value, ns []string) []cli.Flag {
	var names []string

	if name := field.Tag.Get("short"); name != "" {
		names = append(names, strings.Join(append(ns, name), "-"))
	}
	if name := field.Tag.Get("long"); name != "" {
		names = append(names, strings.Join(append(ns, name), "-"))
	}

	if len(names) == 0 {
		return []cli.Flag{}
	}

	flag := cli.GenericFlag{
		Name: strings.Join(names, ", "),
		Value: StructFieldValue{
			field: field,
			value: fieldValue,
		},
		Usage: field.Tag.Get("description"),
		EnvVar: field.Tag.Get("env"),
	}
	return []cli.Flag{StructFieldFlag{GenericFlag: flag}}
}

func getFlagsForStructField(field reflect.StructField, fieldValue reflect.Value, ns []string) []cli.Flag {
	if !fieldValue.IsValid() {
		return []cli.Flag{}
	}

	switch field.Type.Kind() {
	case reflect.Struct:
		if newNs := field.Tag.Get("namespace"); newNs != "" {
			return getFlagsForValue(fieldValue, append(ns, newNs))
		} else if field.Anonymous {
			return getFlagsForValue(fieldValue, ns)
		}
		break
	case reflect.Ptr:
		if field.Type.Elem().Kind() == reflect.Struct {
			if newNs := field.Tag.Get("namespace"); newNs != "" {
				return getFlagsForValue(fieldValue, append(ns, newNs))
			}
		} else {
			return getStructFieldFlag(field, fieldValue, ns)
		}
		break
	case reflect.Chan:
	case reflect.Func:
	case reflect.Interface:
	case reflect.UnsafePointer:
		break
	default:
		return getStructFieldFlag(field, fieldValue, ns)
	}

	return []cli.Flag{}
}

func getFlagsForValue(value reflect.Value, ns []string) []cli.Flag {
	var flags []cli.Flag

	if value.Type().Kind() == reflect.Ptr && value.Type().Elem().Kind() == reflect.Struct {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}

		value = reflect.Indirect(value)
	} else if value.Type().Kind() != reflect.Struct {
		return []cli.Flag{}
	}

	valueType := value.Type()
	for i := 0; i < valueType.NumField(); i++ {
		newFlags := getFlagsForStructField(valueType.Field(i), value.Field(i), ns)
		flags = append(flags, newFlags...)
	}

	return flags
}

func GetFlagsFromStruct(data interface{}, ns... string) []cli.Flag {
	return getFlagsForValue(reflect.ValueOf(data), ns)
}
