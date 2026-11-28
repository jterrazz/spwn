package dotfile_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestAFileWhoseLinesEndInCRLFKeepsThemAndComesBackByteForByte(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": {"b": [1]}}`, ".x.json")
	original := "{\r\n  \"mine\": 1\r\n}\r\n"
	b.write(original, 0o600)
	b.apply()
	merged := b.read()
	if strings.Count(merged, "\n") != strings.Count(merged, "\r\n") || !strings.Contains(merged, `"b": [`) {
		t.Errorf("the merge wrote %q; want every line ended by CRLF, as the file's", merged)
	}
	b.off()
	if got := b.read(); got != original {
		t.Errorf("off gave back %q; want %q", got, original)
	}
}

func TestAFileThatMixesLineEndsIsRefusedAndLeft(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".x.json")
	original := "{\r\n  \"mine\": 1\n}\n"
	b.write(original, 0o600)
	if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.Contains(err.Error(), "mixes CRLF and LF line ends") {
		t.Errorf("Apply: %v; want the file refused", err)
	}
	if got := b.read(); got != original {
		t.Errorf("the file holds %q; want it untouched", got)
	}
}

func TestAFileIsRefusedForWhatItIsWhenItHoldsNoObject(t *testing.T) {
	for name, c := range map[string]struct{ text, want string }{
		"an empty file":         {"", "~/.x.json is empty: Machines merges only into a JSON object"},
		"a file of blank lines": {"\n\n", "~/.x.json is empty: Machines merges only into a JSON object"},
		"a byte order mark":     {string([]byte{0xEF, 0xBB, 0xBF}) + "{}\n", "~/.x.json starts with a byte order mark, which strict JSON has not"},
		"no regular file":       {"", "~/.x.json is no regular file"},
	} {
		t.Run(name, func(t *testing.T) {
			b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".x.json")
			if name == "no regular file" {
				if err := os.MkdirAll(b.file, 0o750); err != nil {
					t.Fatal(err)
				}
			} else {
				b.write(c.text, 0o600)
			}
			if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.HasPrefix(err.Error(), c.want) {
				t.Errorf("Apply: %v; want %q", err, c.want)
			}
		})
	}
}

func TestABlockIsRefusedInAFileWhoseLinesEndInCRLF(t *testing.T) {
	b := newBench(t, catalog.WriteBlock, "set a 1\n", ".x.conf")
	original := "set mine 1\r\n"
	b.write(original, 0o600)
	if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.Contains(err.Error(), "ends its lines with CRLF") {
		t.Errorf("Apply: %v; want the file refused", err)
	}
	if got := b.read(); got != original {
		t.Errorf("the file holds %q; want it untouched", got)
	}
}

func TestABlockGoesIntoNoFileWhoseCommentsAreNotHashLines(t *testing.T) {
	b := newBench(t, catalog.WriteBlock, "set a 1\n", ".vimrc")
	b.write("set mine\n", 0o600)
	if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.Contains(err.Error(), "is no file known to take `#` comments") {
		t.Errorf("Apply: %v; want the file refused", err)
	}
	if got := b.read(); got != "set mine\n" {
		t.Errorf("the file holds %q; want it untouched", got)
	}
}

func TestWhatTheOwnerPutsInAnEmptyContainerOfTheFragmentNeverMovesItsFingerprint(t *testing.T) {
	for _, empty := range []string{"{}", "[]"} {
		t.Run(empty, func(t *testing.T) {
			b := newBench(t, catalog.WriteMerge, `{"hooks": `+empty+`}`, ".x.json")
			b.write("{\n  \"mine\": 1\n}\n", 0o600)
			b.apply()
			applied, err := b.applier.Observe(t.Context(), b.setting)
			if err != nil {
				t.Fatal(err)
			}
			his := map[string]string{"{}": "{\n    \"his\": true\n  }", "[]": "[\n    \"his\"\n  ]"}[empty]
			b.write("{\n  \"mine\": 1,\n  \"hooks\": "+his+"\n}\n", 0o600)
			if edited, _ := b.applier.Observe(t.Context(), b.setting); edited.Digest != applied.Digest {
				t.Error("the owner's own value in the container Machines made reads as changed since")
			}
			other := map[string]string{"{}": "[]", "[]": "{}"}[empty]
			b.write("{\n  \"mine\": 1,\n  \"hooks\": "+other+"\n}\n", 0o600)
			if swapped, _ := b.applier.Observe(t.Context(), b.setting); swapped.Digest == applied.Digest {
				t.Error("a container of another shape in its place reads as Machines'")
			}
		})
	}
}
