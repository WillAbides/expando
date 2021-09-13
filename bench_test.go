package expando

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func BenchmarkExpandEnv(b *testing.B) {
	b.Run("no vars", func(b *testing.B) {
		var buf []byte
		var err error
		tmpl := `the quick brown fox jumps over the lazy dog`
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf, err = ExpandEnv(tmpl, buf[:0])
		}
		if err != nil {
			b.Fatal()
		}
	})

	b.Run("escape in default value", func(b *testing.B) {
		var buf []byte
		err := os.Setenv("fox_speed", "quick")
		if err != nil {
			b.Fatal()
		}
		tmpl := `the ${fox_speed|{\\quick\}} brown fox jumps over the lazy dog`
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf, err = ExpandEnv(tmpl, buf[:0])
		}
		if err != nil {
			b.Fatal()
		}
	})

	for _, count := range []int{1, 10, 100, 1_000, 10_000} {
		err := os.Setenv("fox_speed", "quick")
		if err != nil {
			b.Fatal()
		}
		err = os.Setenv("canine_temperament", "lazy")
		if err != nil {
			b.Fatal()
		}
		tmplBase := "the ${fox_speed|slow} ${fox_color|brown} fox jumps over the ${canine_temperament} dog\n"
		wantBase := "the quick brown fox jumps over the lazy dog\n"
		var tmpl string
		var buf, want []byte
		for i := 0; i < count; i++ {
			tmpl += tmplBase
			want = append(want, []byte(wantBase)...)
		}
		b.Run(fmt.Sprintf("%d lines", count), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tmpl)))
			for i := 0; i < b.N; i++ {
				buf, err = ExpandEnv(tmpl, buf[:0])
			}
			if err != nil {
				b.Fatal()
			}
			if !bytes.Equal(want, buf) {
				b.Fatal()
			}
		})
	}
}

func Benchmark_readVarName(b *testing.B) {
	data := `this_is_a_var_name|this is a value} this is some more text`
	var got string
	var n int
	var err error
	b.ReportAllocs()
	b.SetBytes(int64(len(`this_is_a_var_name|`)))
	for i := 0; i < b.N; i++ {
		got, n, err = readVarName(data)
	}
	if got != "this_is_a_var_name" {
		b.Fatal()
	}
	if n != len("this_is_a_var_name|") {
		b.Fatal()
	}
	if err != nil {
		b.Fatal()
	}
}

func Benchmark_readDefaultValue(b *testing.B) {
	b.Run("simple", func(b *testing.B) {
		data := `this is a value} this is some more text`
		var got string
		var n int
		var err error
		b.ReportAllocs()
		b.SetBytes(int64(len(`this is a value}`)))
		for i := 0; i < b.N; i++ {
			got, n, err = readDefaultValue(data)
		}
		if got != "this is a value" {
			b.Fatal()
		}
		if n != len("this is a value}") {
			b.Fatal()
		}
		if err != nil {
			b.Fatal()
		}
	})

	b.Run("with escape", func(b *testing.B) {
		data := `{this\} is a {value\}} this is some more text`
		var got string
		var n int
		var err error
		b.ReportAllocs()
		b.SetBytes(int64(len(`{this\} is a {value\}}`)))
		for i := 0; i < b.N; i++ {
			got, n, err = readDefaultValue(data)
		}
		if got != "{this} is a {value}" {
			b.Fatal()
		}
		if n != len(`{this\} is a {value\}}`) {
			b.Fatal()
		}
		if err != nil {
			b.Fatal()
		}
	})
}
