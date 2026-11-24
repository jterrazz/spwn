package gate

import (
	"errors"
	"testing"
)

func TestCreateOrAdopt_AdoptsTheContainerAConcurrentCallerCreated(t *testing.T) {
	conflict := errors.New(`docker run: exit status 125 (Conflict. The container name "/spwn-gate" is already in use)`)

	err := createOrAdopt(
		func() error { return conflict },
		func() (bool, error) { return true, nil },
	)

	if err != nil {
		t.Fatalf("createOrAdopt() = %v, want nil when the container exists after the failed create", err)
	}
}

func TestCreateOrAdopt_ReportsAFailedCreateThatLeftNoContainer(t *testing.T) {
	failure := errors.New("docker run: exit status 125 (no such image)")

	err := createOrAdopt(
		func() error { return failure },
		func() (bool, error) { return false, nil },
	)

	if !errors.Is(err, failure) {
		t.Fatalf("createOrAdopt() = %v, want %v", err, failure)
	}
}

func TestCreateOrAdopt_SucceedsWithoutReadingExistenceWhenCreateSucceeds(t *testing.T) {
	err := createOrAdopt(
		func() error { return nil },
		func() (bool, error) {
			t.Fatal("exists() read after a successful create")
			return false, nil
		},
	)

	if err != nil {
		t.Fatalf("createOrAdopt() = %v, want nil", err)
	}
}

func TestCreateOrAdopt_AdoptsAContainerThatBecomesInspectableAfterTheConflict(t *testing.T) {
	conflict := errors.New(`docker run: exit status 125 (Conflict. The container name "/spwn-gate" is already in use by container "66b1f1c0")`)
	reads := 0

	err := createOrAdopt(
		func() error { return conflict },
		func() (bool, error) {
			reads++
			return reads >= 3, nil
		},
	)

	if err != nil {
		t.Fatalf("createOrAdopt() = %v after %d reads, want nil once the winner's container is inspectable", err, reads)
	}
}

func TestCreateOrAdopt_ReportsAConflictWhoseContainerNeverAppears(t *testing.T) {
	conflict := errors.New(`docker run: exit status 125 (Conflict. The container name "/spwn-gate" is already in use by container "66b1f1c0")`)

	err := createOrAdopt(
		func() error { return conflict },
		func() (bool, error) { return false, nil },
	)

	if !errors.Is(err, conflict) {
		t.Fatalf("createOrAdopt() = %v, want %v", err, conflict)
	}
}
