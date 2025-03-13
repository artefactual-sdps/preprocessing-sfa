// Code generated by go-enum DO NOT EDIT.
// Version: 0.6.0
// Revision: 919e61c0174b91303753ee3898569a01abb32c97
// Build Date: 2023-12-18T15:54:43Z
// Built By: goreleaser

package enums

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
)

const (
	// SIPTypeDigitizedAIP is a SIPType of type DigitizedAIP.
	SIPTypeDigitizedAIP SIPType = "DigitizedAIP"
	// SIPTypeDigitizedSIP is a SIPType of type DigitizedSIP.
	SIPTypeDigitizedSIP SIPType = "DigitizedSIP"
	// SIPTypeBornDigitalAIP is a SIPType of type BornDigitalAIP.
	SIPTypeBornDigitalAIP SIPType = "BornDigitalAIP"
	// SIPTypeBornDigitalSIP is a SIPType of type BornDigitalSIP.
	SIPTypeBornDigitalSIP SIPType = "BornDigitalSIP"
)

var ErrInvalidSIPType = fmt.Errorf("not a valid SIPType, try [%s]", strings.Join(_SIPTypeNames, ", "))

var _SIPTypeNames = []string{
	string(SIPTypeDigitizedAIP),
	string(SIPTypeDigitizedSIP),
	string(SIPTypeBornDigitalAIP),
	string(SIPTypeBornDigitalSIP),
}

// SIPTypeNames returns a list of possible string values of SIPType.
func SIPTypeNames() []string {
	tmp := make([]string, len(_SIPTypeNames))
	copy(tmp, _SIPTypeNames)
	return tmp
}

// String implements the Stringer interface.
func (x SIPType) String() string {
	return string(x)
}

// IsValid provides a quick way to determine if the typed value is
// part of the allowed enumerated values
func (x SIPType) IsValid() bool {
	_, err := ParseSIPType(string(x))
	return err == nil
}

var _SIPTypeValue = map[string]SIPType{
	"DigitizedAIP":   SIPTypeDigitizedAIP,
	"DigitizedSIP":   SIPTypeDigitizedSIP,
	"BornDigitalAIP": SIPTypeBornDigitalAIP,
	"BornDigitalSIP": SIPTypeBornDigitalSIP,
}

// ParseSIPType attempts to convert a string to a SIPType.
func ParseSIPType(name string) (SIPType, error) {
	if x, ok := _SIPTypeValue[name]; ok {
		return x, nil
	}
	return SIPType(""), fmt.Errorf("%s is %w", name, ErrInvalidSIPType)
}

func (x SIPType) Ptr() *SIPType {
	return &x
}

// MarshalText implements the text marshaller method.
func (x SIPType) MarshalText() ([]byte, error) {
	return []byte(string(x)), nil
}

// UnmarshalText implements the text unmarshaller method.
func (x *SIPType) UnmarshalText(text []byte) error {
	tmp, err := ParseSIPType(string(text))
	if err != nil {
		return err
	}
	*x = tmp
	return nil
}

var errSIPTypeNilPtr = errors.New("value pointer is nil") // one per type for package clashes

// Scan implements the Scanner interface.
func (x *SIPType) Scan(value interface{}) (err error) {
	if value == nil {
		*x = SIPType("")
		return
	}

	// A wider range of scannable types.
	// driver.Value values at the top of the list for expediency
	switch v := value.(type) {
	case string:
		*x, err = ParseSIPType(v)
	case []byte:
		*x, err = ParseSIPType(string(v))
	case SIPType:
		*x = v
	case *SIPType:
		if v == nil {
			return errSIPTypeNilPtr
		}
		*x = *v
	case *string:
		if v == nil {
			return errSIPTypeNilPtr
		}
		*x, err = ParseSIPType(*v)
	default:
		return errors.New("invalid type for SIPType")
	}

	return
}

// Value implements the driver Valuer interface.
func (x SIPType) Value() (driver.Value, error) {
	return x.String(), nil
}

// Set implements the Golang flag.Value interface func.
func (x *SIPType) Set(val string) error {
	v, err := ParseSIPType(val)
	*x = v
	return err
}

// Get implements the Golang flag.Getter interface func.
func (x *SIPType) Get() interface{} {
	return *x
}

// Type implements the github.com/spf13/pFlag Value interface.
func (x *SIPType) Type() string {
	return "SIPType"
}

// Values implements the entgo.io/ent/schema/field EnumValues interface.
func (x SIPType) Values() []string {
	return SIPTypeNames()
}

// SIPTypeInterfaces returns an interface list of possible values of SIPType.
func SIPTypeInterfaces() []interface{} {
	var tmp []interface{}
	for _, v := range _SIPTypeNames {
		tmp = append(tmp, v)
	}
	return tmp
}

// ParseSIPTypeWithDefault attempts to convert a string to a ContentType.
// It returns the default value if name is empty.
func ParseSIPTypeWithDefault(name string) (SIPType, error) {
	if name == "" {
		return _SIPTypeValue[_SIPTypeNames[0]], nil
	}
	if x, ok := _SIPTypeValue[name]; ok {
		return x, nil
	}
	return SIPType(""), fmt.Errorf("%s is not a valid SIPType, try [%s]", name, strings.Join(_SIPTypeNames, ", "))
}

// NormalizeSIPType attempts to parse a and normalize string as content type.
// It returns the input untouched if name fails to be parsed.
// Example:
//
//	"enUM" will be normalized (if possible) to "Enum"
func NormalizeSIPType(name string) string {
	res, err := ParseSIPType(name)
	if err != nil {
		return name
	}
	return res.String()
}
