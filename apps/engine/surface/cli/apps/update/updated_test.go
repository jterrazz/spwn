package appsupdate

import (
	"errors"
	"testing"
)

func TestAMachineAnUpdateFailedOnSaysWhyWithItsTokenWithheld(t *testing.T) {
	u, failed := updatedOn("linux-4", nil, errors.New("fetch https://bot:hunter2@proxy.example/x refused"))
	if !failed || u.Err != "fetch https://withheld@proxy.example/x refused" {
		t.Errorf("update on linux-4 = %+v, %v; want a failure with its credential withheld", u, failed)
	}
}
