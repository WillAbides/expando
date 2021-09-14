# expando

[![godoc](https://pkg.go.dev/badge/github.com/willabides/expando.svg)](https://pkg.go.dev/github.com/willabides/expando)
[![ci](https://github.com/WillAbides/expando/workflows/ci/badge.svg?branch=main&event=push)](https://github.com/WillAbides/expando/actions?query=workflow%3Aci+branch%3Amain+event%3Apush)

<!--- start godoc --->
Package expando deals with expanding environment variables in template text. Its functions Expand and ExpandEnv are
roughly equivalent to os.Expand and os.ExpandEnv with a few differences in usage. The most prominent difference is
that expando allows default values to be set in the text in the form of ${FOO|default value}.

## Functions

### func [Expand](/expando.go#L40)

`func Expand(tmpl string, lookupEnv Environment, buf []byte) ([]byte, error)`

Expand replaces variables formatted like ${var} or ${var|default value} in tmpl based on values returned by
envLookup. You can use "$$" when your configuration needs a literal dollar sign. When a default value is set like
${var|foo} and there is no mapped value for "var", it will be replaced with "foo". .In a default value, the
character "}" must be escaped with "\}" and the character "\" must be escaped with "\\".
Variable names must start with [a-zA-Z]. Subsequent characters must be [a-zA-Z0-9_].
The result is appended to buf

```golang
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
```

 Output:

```
the quick brown fox jumps over the lazy dog

This is a literal dollar sign: $

This variable's default value contains a backslash and a closing curly bracket: hello\beautiful {world}

You should not escape a dollar sign in a default value: $3.50

You also shouldn't escape a } or a \ outside of a default value.
```

### func [ExpandEnv](/expando.go#L12)

`func ExpandEnv(tmpl string, buf []byte) ([]byte, error)`

ExpandEnv is a shortcut for Expand(tmpl, OSEnv, buf)
<!--- end godoc --->
