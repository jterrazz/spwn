package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/settings"
	"github.com/jterrazz/studio-station/src/settings/contract"
)

const readBudget = 5 * time.Second

const (
	tokenName = catalog.ClaudeTokenName
	kept      = "kept by the studio credential store"
)

type ShareList interface {
	// value: the accounts share list, as the accounts family edits it.
	Get() ([]string, error)
	Add(name string) error
	Remove(name string) error
}

type ClaudeToken struct {
	run   local.Runner
	share ShareList
}

var _ contract.Applier = (*ClaudeToken)(nil)

func NewClaudeToken(run local.Runner, share ShareList) *ClaudeToken {
	return &ClaudeToken{run: run, share: share}
}

func (*ClaudeToken) Kind() catalog.EffectKind { return catalog.ProcClaudeToken.Kind() }

func (c *ClaudeToken) Observe(ctx context.Context, it catalog.Setting) (settings.Observation, error) {
	// invariant: the values name where the token is, never the token.
	now := c.values(ctx)
	o := settings.Observation{State: catalog.StateAbsent, Before: now, After: settings.Sets(it)}
	if now[0].Value == kept {
		o.State, o.Detail = catalog.StateOK, kept
	}
	return o, nil
}

func (c *ClaudeToken) values(ctx context.Context) []settings.KeyValue {
	token := "not kept"
	if _, ok := c.run.LookPath("studio"); ok {
		if _, ok := local.Read(ctx, c.run, local.Cmd{Name: "studio", Args: []string{"credentials", "export", tokenName}, Timeout: readBudget}); ok {
			token = kept
		}
	}
	list, _ := c.share.Get()
	held := "does not hold it"
	if slices.Contains(list, tokenName) {
		held = "holds it"
	}
	return []settings.KeyValue{{Key: "token", Value: token}, {Key: "share list", Value: held}}
}

func (c *ClaudeToken) Apply(ctx context.Context, it catalog.Setting, values map[string]string) (settings.Change, error) {
	// invariant: the token travels on studio's stdin, never in argv.
	if _, ok := c.run.LookPath("studio"); !ok {
		return settings.Change{}, &settings.StudioMissingError{Item: it.Name}
	}
	payload, err := json.Marshal(map[string]string{tokenName: values["token"]})
	if err != nil {
		return settings.Change{}, err
	}
	cmd := local.Command("studio", "credentials", "import")
	cmd.Stdin = string(payload)
	if _, err = local.Must(ctx, c.run, cmd); err != nil {
		return settings.Change{}, fmt.Errorf("studio credentials import failed: %w", err)
	}
	change := settings.Change{Steps: []settings.ChangeStep{{Label: tokenName, State: catalog.StateOK, Detail: kept}}}
	if err := c.share.Add(tokenName); err != nil {
		return change, err
	}
	change.Steps = append(change.Steps, settings.ChangeStep{Label: "share list", State: catalog.StateOK, Detail: "holds " + tokenName})
	return change, nil
}

func (c *ClaudeToken) Undo(ctx context.Context, it catalog.Setting) (settings.Change, error) {
	if _, ok := c.run.LookPath("studio"); !ok {
		return settings.Change{}, &settings.StudioMissingError{Item: it.Name}
	}
	if _, err := local.Must(ctx, c.run, local.Command("studio", "credentials", "delete", tokenName)); err != nil {
		return settings.Change{}, err
	}
	change := settings.Change{Steps: []settings.ChangeStep{{Label: tokenName, State: catalog.StateOK, Detail: "deleted from the studio credential store"}}}
	if err := c.share.Remove(tokenName); err != nil {
		return change, err
	}
	change.Steps = append(change.Steps, settings.ChangeStep{Label: "share list", State: catalog.StateOK, Detail: "no longer holds " + tokenName})
	return change, nil
}
