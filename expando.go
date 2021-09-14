// Package expando deals with expanding environment variables in template text. Its functions Expand and ExpandEnv are
// roughly equivalent to os.Expand and os.ExpandEnv with a few differences in usage. The most prominent difference is
// that expando allows default values to be set in the text in the form of ${FOO|default value}.
package expando

import (
	"fmt"
	"os"
)

// ExpandEnv is a shortcut for Expand(tmpl, OSEnv, buf)
func ExpandEnv(tmpl string, buf []byte) ([]byte, error) {
	return Expand(tmpl, OSEnv, buf)
}

// OSEnv is an Environment that uses os.Lookup to resolve environment variables
var OSEnv envFunc = os.LookupEnv

// Environment is a provider of environment variables
type Environment interface {
	// LookupEnv is equivalent to os.LookupEnv
	LookupEnv(string) (string, bool)
}

// MapEnvironment is an environment provider based on a map
type MapEnvironment map[string]string

// LookupEnv implements Environment.LookupEnv
func (m MapEnvironment) LookupEnv(key string) (string, bool) {
	val, ok := m[key]
	return val, ok
}

// Expand replaces variables formatted like ${var} or ${var|default value} in tmpl based on values returned by
// envLookup. You can use "$$" when your configuration needs a literal dollar sign. When a default value is set like
// ${var|foo} and there is no mapped value for "var", it will be replaced with "foo". .In a default value, the
// character "}" must be escaped with "\}" and the character "\" must be escaped with "\\".
// Variable names must start with [a-zA-Z]. Subsequent characters must be [a-zA-Z0-9_].
// The result is appended to buf
func Expand(tmpl string, lookupEnv Environment, buf []byte) ([]byte, error) {
	i := 0
	dollar := false
	for j := 0; j < len(tmpl); j++ {
		switch tmpl[j] {
		case '$':
			if !dollar {
				dollar = true
				continue
			}
			dollar = false
			buf = append(buf, tmpl[i:j]...)
			i = j + 1
		case '{':
			if !dollar {
				break
			}
			if buf == nil {
				buf = make([]byte, 0, 2*len(tmpl))
			}
			buf = append(buf, tmpl[i:j-1]...)
			name, defaultValue, w, err := varInfo(tmpl[j+1:])
			if err != nil {
				errStringEnd := j + w + 5
				if errStringEnd > len(tmpl) {
					errStringEnd = len(tmpl)
				}
				err = &invalidSyntaxErr{
					position: w + 2,
					value:    tmpl[j-1 : errStringEnd],
					err:      err,
				}
				return nil, err
			}
			val, ok := lookupEnv.LookupEnv(name)
			if ok {
				buf = append(buf, val...)
			} else {
				buf = append(buf, defaultValue...)
			}
			j += w
			i = j + 1
			dollar = false
		default:
			dollar = false
		}
	}
	buf = append(buf, tmpl[i:]...)
	return buf, nil
}

// varInfo returns information about a variable to be expanded.
// data is the remainder of a string after "${"
// name is the variable name
// defaultValue is the default value (the portion after a | pipe) or "" if no pipe is found
// n is the position in data after "}", or in case of an error, it's the position where the syntax becomes invalid
func varInfo(data string) (name, defaultValue string, n int, _ error) {
	var err error
	var nameLen int
	name, nameLen, err = readVarName(data)
	if err != nil {
		return "", "", nameLen, err
	}
	if data[nameLen-1] == '}' {
		return name, "", nameLen, nil
	}
	var valLen int
	defaultValue, valLen, err = readDefaultValue(data[nameLen:])
	if err != nil {
		return "", "", nameLen + valLen, err
	}
	return name, defaultValue, nameLen + valLen, nil
}

// readVarName returns the variable name at the start of data. data should always be a string starting with the
// character immediately after "${". It also returns the number of bytes read.
func readVarName(data string) (string, int, error) {
	if data == "" {
		return "", 0, errUnterminated
	}
	if data[0] == '}' || data[0] == '|' {
		return "", 0, errEmptyString
	}
	if !validNameFirstChar(data[0]) {
		return "", 0, errInvalidStartingCharacter
	}
	i := 1
	for ; i < len(data); i++ {
		c := data[i]
		if c == '}' || c == '|' {
			return data[:i], i + 1, nil
		}
		if !validNameChar(c) {
			return "", i, errInvalidCharacter
		}
	}
	return "", len(data), errUnterminated
}

// readDefaultValue returns a default value. If we are working with text that contains "${foo|bar}", then "|bar}" will
// be passed to readDefaultValue. It also returns the number of bytes read.
func readDefaultValue(data string) (string, int, error) {
	var i int

	// iterate until we find either an escape or a terminator
	// this lets us avoid making buffer if there is no escape character before the terminator
	var hasEscape bool
	for ; i < len(data); i++ {
		switch data[i] {
		case '\\':
			hasEscape = true
		case '}':
			return data[:i], i + 1, nil
		}
		if hasEscape {
			break
		}
	}

	// it's unterminated if we made it to the end without finding a "}"
	if i == len(data) {
		return "", i, errUnterminated
	}

	buf := make([]byte, i, len(data))
	copy(buf, data[:i])
	var escaped, foundClose bool
iter:
	for ; i < len(data); i++ {
		switch data[i] {
		case '\\':
			if escaped {
				buf = append(buf, data[i])
			}
			escaped = !escaped
			continue
		case '}':
			if !escaped {
				foundClose = true
				break iter
			}
		}
		if escaped && data[i] != '}' {
			return "", i, errInvalidEscape
		}
		buf = append(buf, data[i])
		escaped = false
	}
	if !foundClose {
		return "", i, errUnterminated
	}
	return string(buf), i + 1, nil
}

type invalidSyntaxErr struct {
	position int
	value    string
	err      error
}

func (e *invalidSyntaxErr) Error() string {
	return fmt.Sprintf(
		"invalid syntax at position %d of %q: %v",
		e.position, e.value, e.err,
	)
}

var (
	errInvalidCharacter         = fmt.Errorf("invalid character")
	errInvalidStartingCharacter = fmt.Errorf("invalid starting character")
	errUnterminated             = fmt.Errorf("unterminated")
	errEmptyString              = fmt.Errorf("empty string")
	errInvalidEscape            = fmt.Errorf("invalid escape sequence")
)

func validNameFirstChar(c uint8) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_'
}

func validNameChar(c uint8) bool {
	return validNameFirstChar(c) || '0' <= c && c <= '9'
}

type envFunc func(string) (string, bool)

func (fn envFunc) LookupEnv(key string) (string, bool) {
	return fn(key)
}
