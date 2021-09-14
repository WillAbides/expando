//go:build gofuzzbeta

package expando

import (
	"regexp"
	"strings"
	"testing"

	"github.com/matryer/is"
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
	require := is.New(t)
	require.Helper()

	val, valLen, err := readDefaultValue(data)
	if err == nil {
		require.True(len(val) < valLen)
	}
	if len(val) > 0 {
		require.True(strings.HasPrefix(
			stripChars(data, `\}`),
			stripChars(val, `\}`),
		))
	}
}

func testVarInfoProperties(t *testing.T, data string) {
	require := is.New(t)
	if data == "" {
		return
	}
	name, defaultValue, w, err := varInfo(data)
	_, _ = name, defaultValue
	switch err {
	case errUnterminated:
		require.True(!regexp.MustCompile(`[^\\]}`).MatchString(data))
		require.True(data[0] != '}')
	case errEmptyString:
		require.True(data[0] == '}' || data[0] == '|')
	case errInvalidStartingCharacter:
		require.True(!validNameFirstChar(data[0]))
	case errInvalidCharacter:
		require.True(!validNameChar(data[w]))
	case nil:
		require.True(len(name)+len(defaultValue) < w)
		require.True(data[w-1] == '}')
		require.True(regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name))
	}
}

func testReadVarNameProperties(t *testing.T, data string) {
	require := is.New(t)
	t.Helper()
	name, nameLen, err := readVarName(data)
	switch err {
	case nil:
		require.True(len(name) < nameLen)
		require.True(data[nameLen-1] == '}' || data[nameLen-1] == '|')
		require.True(regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name))
	case errEmptyString:
		require.True(data[0] == '}' || data[0] == '|')
	case errInvalidStartingCharacter:
		require.True(!validNameFirstChar(data[0]))
	case errInvalidCharacter:
		require.True(!validNameChar(data[nameLen]))
	case errUnterminated:
		require.True(!strings.Contains(data, "}"))
		require.True(!strings.Contains(data, "|"))
	}
	if len(name) > 0 {
		require.True(strings.HasPrefix(data, name))
	}
}

func Test_parseEnv(t *testing.T) {
	require := is.New(t)
	got := parseEnv(`
FOO=bar
 BAZ 	=qux
asdf

`)
	require.Equal(MapEnvironment{
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
