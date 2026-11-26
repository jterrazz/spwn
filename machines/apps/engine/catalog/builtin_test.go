package catalog_test

import (
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// invariant: the compiled table is the strict read of the embedded files,
// value for value: a package file the decoder or the whole-catalog judge
// refuses fails here, never in a user's run.
func TestTheCompiledCatalogIsTheStrictReadOfItsFiles(t *testing.T) {
	read, err := catalog.Load(catalog.BuiltinFiles(), "packages")
	if err != nil {
		t.Fatal(err)
	}
	compiled := builtin(t)
	if !reflect.DeepEqual(compiled, read) {
		t.Fatal("the compiled built-in catalog differs from its files: run make generate and commit the result")
	}
}

func TestTheBuiltInCatalogDecodesNoFile(t *testing.T) {
	before := catalog.Decoded()
	builtin(t)
	if n := catalog.Decoded() - before; n != 0 {
		t.Errorf("the built-in catalog decoded %d files, want none", n)
	}
}

func BenchmarkBuiltin(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := catalog.Builtin(); err != nil {
			b.Fatal(err)
		}
	}
}

// value: the strict read the table is compiled from, what every start paid
// before it was.
func BenchmarkLoadFromFiles(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := catalog.Load(catalog.BuiltinFiles(), "packages"); err != nil {
			b.Fatal(err)
		}
	}
}
