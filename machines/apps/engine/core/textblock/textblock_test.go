package textblock

import "testing"

func TestAHandEditedBlockReadsDifferentAndADamagedOneIsToldApart(t *testing.T) {
	if body, state := Of("# >>> station:a >>>\nmine\n# <<< station:a <<<\n", "station:a"); state != Found || body != "mine\n" {
		t.Errorf("read %q, %v", body, state)
	}
	if _, state := Of("# >>> station:a >>>\nno end\n", "station:a"); state != Damaged {
		t.Errorf("a start with no end reads %v", state)
	}
	twice := "# >>> station:a >>>\n1\n# <<< station:a <<<\n# >>> station:a >>>\n2\n# <<< station:a <<<\n"
	if _, state := Of(twice, "station:a"); state != Damaged {
		t.Errorf("a block twice reads %v", state)
	}
	if _, state := Of("export EDITOR=vim\n", "station:a"); state != Absent {
		t.Errorf("no block reads %v", state)
	}
}

func TestAStartMetAgainBeforeItsEndIsDamage(t *testing.T) {
	stray := "mine\n# >>> station:a >>>\nmine too\n# >>> station:a >>>\nk\n# <<< station:a <<<\n"
	if _, state := Of(stray, "station:a"); state != Damaged {
		t.Fatalf("a stray start above a block = %v; want damaged", state)
	}
	if got := DamagedLines(stray, "station:a"); len(got) != 1 || got[0] != "line 2" {
		t.Fatalf("DamagedLines = %q; want the stray start alone", got)
	}
	if first, last := Lines(stray, "station:a"); first != 4 || last != 6 {
		t.Fatalf("Lines = %d, %d; want the whole block, 4 to 6", first, last)
	}
	twice := "# >>> station:a >>>\nk\n# <<< station:a <<<\n# >>> station:a >>>\nk\n# <<< station:a <<<\n"
	if got := DamagedLines(twice, "station:a"); len(got) != 1 || got[0] != "lines 4 to 6" {
		t.Fatalf("DamagedLines = %q; want the second block", got)
	}
}

func TestTheGuestsBlockIsABlockAndTheMainBlockIsNot(t *testing.T) {
	if body, state := Of("mine\n# >>> station hosts >>>\nk\n# <<< station hosts <<<\n", "station hosts"); state != Found || body != "k\n" {
		t.Errorf("the guest's block reads %q, %v", body, state)
	}
	if spans := Spans([]string{"# >>> station >>>", "x", "# <<< station <<<"}); len(spans) != 0 {
		t.Errorf("the main block reads as %+v", spans)
	}
}
