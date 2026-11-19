package calls

import (
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Deps struct {
	SignIns contract.SignIns
	Kind    machine.Kind
}

func All(d Deps) []call.Handler {
	return []call.Handler{
		call.BindWatch(Programs, d), call.BindWatch(Login, d), call.Bind(Logout, d), call.Bind(Open, d), call.Bind(Answer, d),
	}
}
