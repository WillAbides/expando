//go:build gofuzzbeta

package expando

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Fuzz(f *testing.F) {
	f.Add(`
${FOO}
${BAZ|omg}
${BAR}
${BAR|this is a default value}
`, `
FOO=bar
BAZ=qux
`)
	f.Add("asdf}jkl;", "")
	f.Add("asdf}", "")
	f.Add(`asdf\}`, "")
	f.Add("asdf|default value}jkl;", "")
	f.Fuzz(func(t *testing.T, data, env string) {
		fuzzExpand(t, data, env)
		testVarInfoProperties(t, data)
		testReadVarNameProperties(t, data)
		testReadDefaultValueProperties(t, data)
	})
}

func fuzzExpand(t *testing.T, tmpl, env string) {
	// nolint:errcheck // we are just checking for panics
	_, _ = Expand(tmpl, parseEnv(env), nil)
}

func testReadDefaultValueProperties(t *testing.T, data string) {
	t.Helper()

	val, valLen, err := readDefaultValue(data)
	if err == nil {
		require.True(t, len(val) < valLen)
	}
	if len(val) > 0 {
		require.True(t, strings.HasPrefix(
			stripChars(data, `\}`),
			stripChars(val, `\}`),
		))
	}
}

func testVarInfoProperties(t *testing.T, data string) {
	if data == "" {
		return
	}
	name, defaultValue, w, err := varInfo(data)
	_, _ = name, defaultValue
	switch err {
	case errUnterminated:
		require.True(t, !regexp.MustCompile(`[^\\]}`).MatchString(data))
		require.True(t, data[0] != '}')
	case errEmptyString:
		require.True(t, data[0] == '}' || data[0] == '|')
	case errInvalidStartingCharacter:
		require.True(t, !validNameFirstChar(data[0]))
	case errInvalidCharacter:
		require.True(t, !validNameChar(data[w]))
	case nil:
		require.True(t, len(name)+len(defaultValue) < w)
		require.True(t, data[w-1] == '}')
		require.True(t, regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name))
	}
}

func testReadVarNameProperties(t *testing.T, data string) {
	t.Helper()
	name, nameLen, err := readVarName(data)
	switch err {
	case nil:
		require.True(t, len(name) < nameLen)
		require.True(t, data[nameLen-1] == '}' || data[nameLen-1] == '|')
		require.True(t, regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name))
	case errEmptyString:
		require.True(t, data[0] == '}' || data[0] == '|')
	case errInvalidStartingCharacter:
		require.True(t, !validNameFirstChar(data[0]))
	case errInvalidCharacter:
		require.True(t, !validNameChar(data[nameLen]))
	case errUnterminated:
		require.True(t, !strings.Contains(data, "}"))
		require.True(t, !strings.Contains(data, "|"))
	}
	if len(name) > 0 {
		require.True(t, strings.HasPrefix(data, name))
	}
}

func Test_parseEnv(t *testing.T) {
	got := parseEnv(`
FOO=bar
 BAZ 	=qux
asdf

`)
	require.Equal(t, MapEnvironment{
		"FOO": "bar",
		"BAZ": "qux",
	}, got)
}

func parseEnv(input string) MapEnvironment {
	output := MapEnvironment{}
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		output[key] = parts[1]
	}
	return output
}

func stripChars(data, chars string) string {
	for i := range chars {
		data = strings.ReplaceAll(data, string(chars[i]), "")
	}
	return data
}
