package accounts

import "github.com/jterrazz/studio-station/apps/engine/core/fail"

const (
	LoginUnknownProgram  fail.Code = "login.unknown-program"
	LoginNotInstalled    fail.Code = "login.not-installed"
	LoginAlreadySignedIn fail.Code = "login.already-signed-in"
	LoginMachineStopped  fail.Code = "login.machine-stopped"
	LoginBusy            fail.Code = "login.busy"
	LoginSiteRefused     fail.Code = "login.site-refused"
	LoginExpired         fail.Code = "login.expired"
	LoginFailed          fail.Code = "login.failed"
	LoginNotSignedIn     fail.Code = "login.not-signed-in"
	LoginGranted         fail.Code = "login.granted"
	LoginNoExchange      fail.Code = "login.no-exchange"
	LoginOtherMac        fail.Code = "login.other-mac"
	LoginNotAsking       fail.Code = "login.not-asking"
	LoginCodeRefused     fail.Code = "login.code-refused"
)

const (
	VaultLabelTaken    fail.Code = "vault.label-taken"
	VaultEmptyLabel    fail.Code = "vault.empty-label"
	VaultEmptyValue    fail.Code = "vault.empty-value"
	VaultNotSignedIn   fail.Code = "vault.not-signed-in"
	VaultCannotCapture fail.Code = "vault.cannot-capture"
	VaultUnknown       fail.Code = "vault.unknown"
)

const (
	GrantUnknownEntry      fail.Code = "grant.unknown-entry"
	GrantUnknownMachine    fail.Code = "grant.unknown-machine"
	GrantUnknownAgent      fail.Code = "grant.unknown-agent"
	GrantToServer          fail.Code = "grant.to-server"
	GrantOtherMac          fail.Code = "grant.other-mac"
	GrantMachineStopped    fail.Code = "grant.machine-stopped"
	GrantSignedInHere      fail.Code = "grant.signed-in-here"
	GrantNotAgentGrantable fail.Code = "grant.not-agent-grantable"
	GrantPresenceRefused   fail.Code = "grant.presence-refused"
	GrantNoPresence        fail.Code = "grant.no-presence"
	GrantCannotDeliver     fail.Code = "grant.cannot-deliver"
	GrantNotGranted        fail.Code = "grant.not-granted"
	GrantNoPassword        fail.Code = "grant.no-password"
	GrantManyPasswords     fail.Code = "grant.many-passwords"
)
