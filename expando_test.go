package expando

import (
	"fmt"
	"testing"

	"github.com/matryer/is"
)

func ExampleExpand() {
	env := MapEnvironment{
		"fox_speed":          "quick",
		"canine_temperament": "lazy",
	}

	tmpl := `the ${fox_speed} ${fox_color|brown} fox jumps over the ${canine_temperament|alert} dog

This is a literal dollar sign: $$

This variable's default value contains a backslash and a closing curly bracket: ${FAKE_VAR|hello\\beautiful {world\}}

You should not escape a dollar sign in a default value: ${FAKE_VAR|$3.50}

You also shouldn't escape a } or a \ outside of a default value.`

	output, err := Expand(tmpl, env, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(output))

	// output: the quick brown fox jumps over the lazy dog
	//
	// This is a literal dollar sign: $
	//
	// This variable's default value contains a backslash and a closing curly bracket: hello\beautiful {world}
	//
	// You should not escape a dollar sign in a default value: $3.50
	//
	// You also shouldn't escape a } or a \ outside of a default value.
}

func Test_varInfo(t *testing.T) {
	for _, td := range []struct {
		input        string
		name         string
		defaultValue string
		length       int
		wantErr      bool
	}{
		{input: `foo}`, name: `foo`, length: 4},
		{input: `f}`, name: `f`, length: 2},
		{input: `}`, wantErr: true},
		{input: `foo`, length: 3, wantErr: true},
		{input: `H}OME`, name: `H`, length: 2, wantErr: false},
		{input: `foo|bar}`, name: `foo`, defaultValue: `bar`, length: 8},
		{input: `foo|hello|world}`, name: `foo`, defaultValue: `hello|world`, length: 16},
		{input: `w\orld}`, wantErr: true, length: 1},
		{input: `foo|}`, name: `foo`, length: 5},
	} {
		t.Run(td.input, func(t *testing.T) {
			require := is.New(t)
			name, defaultValue, length, err := varInfo(td.input)
			if td.wantErr {
				require.True(err != nil)
			} else {
				require.NoErr(err)
			}
			require.Equal(td.name, name)
			require.Equal(td.defaultValue, defaultValue)
			require.Equal(td.length, length)
		})
	}
}

func Test_readDefaultValue(t *testing.T) {
	for _, td := range []struct {
		input  string
		output string
		length int
		err    error
	}{
		{input: "", err: errUnterminated},
		{input: `asdf`, length: 4, err: errUnterminated},
		{input: `asdf}`, length: 5, output: `asdf`},
		{input: `as\}f}`, length: 6, output: `as}f`},
		{input: `as\}}f}`, length: 5, output: `as}`},
		{input: `as\\\}}f}`, length: 7, output: `as\}`},
		{input: `\\\}}f}`, length: 5, output: `\}`},
		{input: `\\\}}f}`, length: 5, output: `\}`},
		{input: `world|foo}`, length: 10, output: `world|foo`},
		{input: `w\orld}`, length: 2, err: errInvalidEscape},
		{input: `world`, length: 5, err: errUnterminated},
		{input: `w\\orld`, length: 7, err: errUnterminated},
	} {
		t.Run(td.input, func(t *testing.T) {
			require := is.New(t)
			output, length, err := readDefaultValue(td.input)
			require.Equal(td.err, err)
			require.Equal(td.length, length)
			require.Equal(td.output, output)
		})
	}
}

func Test_readVarName(t *testing.T) {
	for _, td := range []struct {
		input  string
		output string
		length int
		err    error
	}{
		{input: ``, err: errUnterminated},
		{input: `|`, err: errEmptyString},
		{input: `}asdf`, err: errEmptyString},
		{input: `asdf`, length: 4, err: errUnterminated},
		{input: `as*f}`, length: 2, err: errInvalidCharacter},
		{input: `asdf}jkl;`, length: 5, output: `asdf`},
		{input: `asdf}`, length: 5, output: `asdf`},
		{input: `asdf|jkl;`, length: 5, output: `asdf`},
		{input: `2asdf|jkl;`, err: errInvalidStartingCharacter},
		{input: `{`, err: errInvalidStartingCharacter},
	} {
		t.Run(td.input, func(t *testing.T) {
			require := is.New(t)
			output, length, err := readVarName(td.input)
			require.Equal(td.err, err)
			require.Equal(td.length, length)
			require.Equal(td.output, output)
		})
	}
}

func TestExpand(t *testing.T) {
	lookupEnv := MapEnvironment{
		`HOME`:   `/usr/gopher`,
		`H`:      `(Value of H)`,
		`home_1`: `/usr/foo`,
		`this`:   `that`,
	}
	for _, td := range []struct {
		in  string
		out string
		err error
	}{
		{},
		{in: `$*`, out: `$*`},
		{in: `{${HOME}}`, out: `{/usr/gopher}`},
		{in: `$${this}`, out: `${this}`},
		{in: `$$${this}`, out: `$that`},
		{in: `${HOME|unterminated`, err: newInvalidSyntaxError(19, `${HOME|unterminated`, errUnterminated)},
		{in: `$1`, out: `$1`},
		{in: `${1}`, err: newInvalidSyntaxError(2, `${1}`, errInvalidStartingCharacter)},
		{in: `now is the time`, out: `now is the time`},
		{in: `${home_1}`, out: `/usr/foo`},
		{in: `${H}OME`, out: `(Value of H)OME`},
		{in: `a${H}run`, out: `a(Value of H)run`},
		{in: `start$+middle$^end$`, out: `start$+middle$^end$`},
		{in: `$`, out: `$`},
		{in: `$}`, out: `$}`},
		{in: `${`, err: newInvalidSyntaxError(2, `${`, errUnterminated)},
		{in: `${asdf`, err: newInvalidSyntaxError(6, `${asdf`, errUnterminated)},
		{in: `a$df${asdf`, err: newInvalidSyntaxError(6, `${asdf`, errUnterminated)},
		{in: `${}`, err: newInvalidSyntaxError(2, `${}`, errEmptyString)},
		{in: `abc${}`, err: newInvalidSyntaxError(2, `${}`, errEmptyString)},
		{in: `abc${hello|world|foo}`, out: `abcworld|foo`},
		{in: `abc${hello|`, err: newInvalidSyntaxError(8, `${hello|`, errUnterminated)},
		{in: `abc${hello|w\orld}`, err: newInvalidSyntaxError(10, `${hello|w\orld`, errInvalidEscape)},
		{in: `abc${hello\world}`, err: newInvalidSyntaxError(7, `${hello\wor`, errInvalidCharacter)},
		{in: `${hello|\\world}`, out: `\world`},
	} {
		t.Run(td.in, func(t *testing.T) {
			require := is.New(t)
			result, err := Expand(td.in, lookupEnv, nil)
			if td.err != nil {
				require.Equal(err.Error(), td.err.Error())
			} else {
				require.NoErr(err)
			}
			require.Equal(td.out, string(result))
		})
	}
}

func newInvalidSyntaxError(position int, value string, err error) *invalidSyntaxErr {
	return &invalidSyntaxErr{
		position: position,
		value:    value,
		err:      err,
	}
}
