package settings

import "testing"

const owner = "export EDITOR=vim\n\nalias gs='git status'\n"

func TestANewBlockGoesAfterTheLastStationBlockAndLeavesTheOwnersLinesAlone(t *testing.T) {
	text, _ := UpsertBlock(owner)
	text, changed := UpsertPackageBlock(text, "station:a", "echo a\n")
	if !changed {
		t.Fatal("a new block reads unchanged")
	}
	text, _ = UpsertPackageBlock(text, "station:b/path", "echo b\n")
	want := owner + "\n" + block + "# >>> station:a >>>\necho a\n# <<< station:a <<<\n# >>> station:b/path >>>\necho b\n# <<< station:b/path <<<\n"
	if text != want {
		t.Errorf("got\n%s\nwant\n%s", text, want)
	}
}

func TestABlockIsReplacedInPlaceBetweenTheOwnersLines(t *testing.T) {
	text := "top\n# >>> station:a >>>\nold\n# <<< station:a <<<\nmiddle\n# >>> station:b >>>\nb\n# <<< station:b <<<\nbottom\n"
	got, changed := UpsertPackageBlock(text, "station:a", "new\nlines\n")
	want := "top\n# >>> station:a >>>\nnew\nlines\n# <<< station:a <<<\nmiddle\n# >>> station:b >>>\nb\n# <<< station:b <<<\nbottom\n"
	if !changed || got != want {
		t.Errorf("got %q, %v; want %q", got, changed, want)
	}
	if again, changed := UpsertPackageBlock(got, "station:a", "new\nlines\n"); changed || again != got {
		t.Errorf("an upsert of the same body changed the file: %q", again)
	}
}

func TestRemovingABlockGivesTheFileBackAsItWas(t *testing.T) {
	for _, before := range []string{"", owner, "no newline at the end", "\n\n", owner + "\n# >>> station >>>\nx\n# <<< station <<<\n", "# >>> station >>>\nx\n# <<< station <<<"} {
		written, _ := UpsertPackageBlock(before, "station:a", "echo a\n")
		back, changed := RemovePackageBlock(written, "station:a")
		if !changed || back != before {
			t.Errorf("removed from %q leaves %q, %v; want %q", written, back, changed, before)
		}
	}
	if same, changed := RemovePackageBlock(owner, "station:a"); changed || same != owner {
		t.Errorf("removing a block that is not there changed the file")
	}
}

func TestAMainBlockAddedLaterGoesBeforeThePackageBlocks(t *testing.T) {
	text, _ := UpsertPackageBlock("mine\n", "station:a", "echo a\n")
	text, changed := UpsertBlock(text)
	want := "mine\n\n" + block + "# >>> station:a >>>\necho a\n# <<< station:a <<<\n"
	if !changed || text != want {
		t.Errorf("got %q, %v; want %q", text, changed, want)
	}
	if again, changed := UpsertBlock(text); changed || again != text {
		t.Errorf("a second init changed the file: %q", again)
	}
}

func TestAHandEditedBlockReadsDifferentAndADamagedOneIsToldApart(t *testing.T) {
	if body, state := PackageBlock("# >>> station:a >>>\nmine\n# <<< station:a <<<\n", "station:a"); state != BlockHere || body != "mine\n" {
		t.Errorf("read %q, %v", body, state)
	}
	if _, state := PackageBlock("# >>> station:a >>>\nno end\n", "station:a"); state != BlockDamaged {
		t.Errorf("a start with no end reads %v", state)
	}
	twice := "# >>> station:a >>>\n1\n# <<< station:a <<<\n# >>> station:a >>>\n2\n# <<< station:a <<<\n"
	if _, state := PackageBlock(twice, "station:a"); state != BlockDamaged {
		t.Errorf("a block twice reads %v", state)
	}
	if _, state := PackageBlock(owner, "station:a"); state != BlockAbsent {
		t.Errorf("no block reads %v", state)
	}
}

func TestPackageBlockIDsListTheBlocksInFileOrder(t *testing.T) {
	text, _ := UpsertPackageBlock(owner, "station:b", "b\n")
	text, _ = UpsertPackageBlock(text, "station:a/x", "a\n")
	ids := PackageBlockIDs(text)
	if len(ids) != 2 || ids[0] != "station:b" || ids[1] != "station:a/x" {
		t.Errorf("listed %v", ids)
	}
}
