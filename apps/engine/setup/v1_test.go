package setup_test

import (
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/setup"
)

func TestConvertV1KeepsTheRegistryAndDropsRolesAndTheSecret(t *testing.T) {
	v1 := `{
  "remote": {"mode": "kernel", "auth_method": "authkey", "secret": "an-old-auth-secret", "hostname": "mbp"},
  "user_email": "a@b.c",
  "self": "macbook",
  "machines": {
    "macbook": {"role": "client"},
    "mac-mini": {"role": "server", "ssh": "jterrazz.agent@mac-mini", "identity": "~/.ssh/id_ed25519"}
  }
}`
	got, err := setup.ConvertV1([]byte(v1), "jterrazz")
	if err != nil {
		t.Fatal(err)
	}
	want := setup.V1Preferences{
		ThisMachine: "macbook",
		Macs: map[string]setup.Mac{
			"macbook":  {},
			"mac-mini": {User: "jterrazz.agent", Key: "~/.ssh/id_ed25519"},
		},
		VMUser:    "jterrazz",
		UserEmail: "a@b.c",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("converted\n%+v\nwant\n%+v", got, want)
	}
}

func TestConvertV1NamesThisMacFromTheHostname(t *testing.T) {
	got, err := setup.ConvertV1([]byte(`{"remote": {"hostname": "mac-mini"}}`), "agent")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Macs["mac-mini"]; got.ThisMachine != "mac-mini" || !ok {
		t.Fatalf("converted %+v", got)
	}
}

func TestConvertV1RefusesWhatIsNotJSON(t *testing.T) {
	if _, err := setup.ConvertV1([]byte("{"), "u"); err == nil {
		t.Fatal("a broken config.json converted")
	}
}
