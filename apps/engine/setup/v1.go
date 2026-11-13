package setup

import (
	"encoding/json"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

type v1Document struct {
	Remote struct {
		Hostname string `json:"hostname"`
	} `json:"remote"`
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name"`
	Self      string `json:"self"`
	Machines  map[string]struct {
		SSH      string `json:"ssh"`
		Identity string `json:"identity"`
	} `json:"machines"`
}

func ConvertV1(v1 []byte, user string) (V1Preferences, error) {
	// invariant: the v1 tailnet mode and auth key are dropped: the Tailscale app
	// logs in itself, and station keeps no tailnet secret.
	var old v1Document
	if err := json.Unmarshal(v1, &old); err != nil {
		return V1Preferences{}, &store.UnreadableError{Path: "config.json", Cause: err}
	}
	var s V1Preferences
	s.ThisMachine = old.Self
	s.VMUser = user
	s.UserEmail = old.UserEmail
	s.UserName = old.UserName
	for name, m := range old.Machines {
		if s.Macs == nil {
			s.Macs = map[string]Mac{}
		}
		login, _, found := strings.Cut(m.SSH, "@")
		if !found {
			login = ""
		}
		s.Macs[name] = Mac{User: login, Key: m.Identity}
	}
	if s.ThisMachine == "" && old.Remote.Hostname != "" {
		s.ThisMachine = old.Remote.Hostname
	}
	if s.ThisMachine != "" {
		if s.Macs == nil {
			s.Macs = map[string]Mac{}
		}
		if _, ok := s.Macs[s.ThisMachine]; !ok {
			s.Macs[s.ThisMachine] = Mac{}
		}
	}
	return s, nil
}
