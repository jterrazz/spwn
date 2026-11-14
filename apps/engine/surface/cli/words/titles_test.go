package words_test

import (
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
)

func TestACodeIsCalledByItsNameReadAsWordsAndItsFamily(t *testing.T) {
	for code, want := range map[fail.Code][2]string{
		"machines.too-little-memory": {"too little memory", "machines"},
		"port.on-another-mac":        {"on another Mac", "port"},
		"account.unknown-vm":         {"unknown VM", "account"},
		"guest_timeout":              {"Guest did not answer", ""},
		"unknown_vm":                 {"Unknown machine", ""},
		"vm_stopped":                 {"Machine stopped", ""},
	} {
		if name, family := words.Title(code); name != want[0] || family != want[1] {
			t.Errorf("%s is called %q of %q; want %q of %q", code, name, family, want[0], want[1])
		}
	}
}

func TestEveryCodeIsListedOnceInOneOrder(t *testing.T) {
	listed := words.CodesSorted()
	if len(listed) != len(words.Codes()) {
		t.Fatalf("%d codes listed of %d worded", len(listed), len(words.Codes()))
	}
	for i := 1; i < len(listed); i++ {
		if listed[i-1] >= listed[i] {
			t.Errorf("%s comes before %s", listed[i-1], listed[i])
		}
	}
}

func TestAnEngineRefusalIsTitledByHandNeverReadFromItsName(t *testing.T) {
	for _, code := range []fail.Code{
		enginecalls.StartFailed, enginecalls.VMStopped, enginecalls.UnknownVM, enginecalls.UnknownOp,
		enginecalls.UnknownPage, enginecalls.GuestAbsent, enginecalls.GuestTimeout, enginecalls.IgnoredPort,
		enginecalls.NotAllowed, enginecalls.RangeFull, enginecalls.AwakeRun, enginecalls.AwakeRange,
		enginecalls.AwakeGuard, enginecalls.AwakeUnsupported, enginecalls.Internal,
	} {
		name, family := words.Title(code)
		first, _ := utf8.DecodeRuneInString(name)
		if family != "" || name == "" || !unicode.IsUpper(first) {
			t.Errorf("%s is titled %q of %q; want a written title with no family", code, name, family)
		}
	}
}
