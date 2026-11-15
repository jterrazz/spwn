package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

const readBudget = 5 * time.Second

const (
	tokenName = catalog.ClaudeTokenName
	kept      = "kept by the studio credential store"
)

type ClaudeToken struct {
	run local.Runner
}

var _ contract.Applier = (*ClaudeToken)(nil)

func NewClaudeToken(run local.Runner) *ClaudeToken {
	return &ClaudeToken{run: run}
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
	return []settings.KeyValue{{Key: "token", Value: token}}
}

func (c *ClaudeToken) Apply(ctx context.Context, it catalog.Setting, values map[string]string) (settings.Change, error) {
	// invariant: the token travels on studio's stdin, never in argv.
	if _, ok := c.run.LookPath("studio"); !ok {
		return settings.Change{}, &settings.StudioMissingError{Package: it.Name}
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
	return settings.Change{Steps: []settings.ChangeStep{{Label: tokenName, State: catalog.StateOK, Detail: kept}}}, nil
}

func (c *ClaudeToken) Undo(ctx context.Context, it catalog.Setting) (settings.Change, error) {
	if _, ok := c.run.LookPath("studio"); !ok {
		return settings.Change{}, &settings.StudioMissingError{Package: it.Name}
	}
	if _, err := local.Must(ctx, c.run, local.Command("studio", "credentials", "delete", tokenName)); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: tokenName, State: catalog.StateOK, Detail: "deleted from the studio credential store"}}}, nil
}
