package architect

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"spwn.sh/packages/activity"
	"spwn.sh/packages/auth"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/world/backend"
	"spwn.sh/packages/world/labels"
)

// DaemonInfo describes the state of the Architect daemon container.
type DaemonInfo struct {
	ContainerID string
	Image       string
	Status      string
	Running     bool
	StartedAt   time.Time
	Uptime      time.Duration
	OrgName     string
}

// StartDaemonOpts configures architect daemon spawn. All fields are
// optional. The OnProgress callback is the canonical place to surface
// real-time spawn diagnostics - both the CLI stepper and the web UI's
// status endpoint feed off it.
type StartDaemonOpts struct {
	// ImageOverride lets the caller pin a specific architect image
	// (used by tests and SPWN_ARCHITECT_IMAGE). When empty, the
	// canonical image is built/refreshed by the image package.
	ImageOverride string
	// LogWriter receives raw output from the image build (npm install,
	// docker build steps, …). nil → io.Discard.
	LogWriter io.Writer
	// OnProgress is called at each step of the spawn pipeline with
	// (event, detail) pairs. Events are stable strings that the
	// the API polls; detail is a free-form human-readable note.
	OnProgress func(event, detail string)
}

func (o *StartDaemonOpts) progress(event, detail string) {
	if o != nil && o.OnProgress != nil {
		o.OnProgress(event, detail)
	}
}

func (o *StartDaemonOpts) writer() io.Writer {
	if o != nil && o.LogWriter != nil {
		return o.LogWriter
	}
	return io.Discard
}

// StartDaemon creates and starts the spwn-architect Docker container.
// It returns the container ID. If the container is already running,
// it returns an error indicating that.
//
// This is the back-compat shim. New code should call
// StartDaemonWithOpts so it can subscribe to progress events.
func StartDaemon(ctx context.Context, imageOverride string, logWriters ...io.Writer) (string, error) {
	var lw io.Writer
	if len(logWriters) > 0 {
		lw = logWriters[0]
	}
	return StartDaemonWithOpts(ctx, StartDaemonOpts{
		ImageOverride: imageOverride,
		LogWriter:     lw,
	})
}

// StartDaemonWithOpts is the rich entry point. It emits an OnProgress
// event at every step so callers can render real-time spawn diagnostics
// instead of guessing from elapsed time.
//
// Event vocabulary (in order):
//
//	docker_check         - opening the Docker client
//	already_running      - fast path, returned with error
//	cleanup              - removing a stopped container
//	image_resolve        - resolving image tag
//	image_building       - building the architect image (long step)
//	image_ready          - image present
//	credentials_sync     - credentials being written
//	host_files           - stack.md / knowledge dir bootstrapped
//	container_creating   - Docker create call
//	container_starting   - Docker start call
//	ready                - daemon up and labelled
func StartDaemonWithOpts(ctx context.Context, opts StartDaemonOpts) (string, error) {
	opts.progress("docker_check", "opening Docker client")
	docker, err := backend.NewDocker()
	if err != nil {
		return "", fmt.Errorf("docker is not reachable: %w", err)
	}

	info, err := docker.Inspect(ctx, platform.ArchitectContainerName())
	if err == nil && info.Running {
		opts.progress("already_running", info.ID)
		return info.ID, fmt.Errorf("architect is already running (container %s)", platform.ArchitectContainerName())
	}

	if err == nil && !info.Running {
		opts.progress("cleanup", "removing stopped architect container")
		_ = docker.Remove(ctx, platform.ArchitectContainerName())
	}

	image := platform.ArchitectImage
	if opts.ImageOverride != "" {
		image = opts.ImageOverride
	}
	opts.progress("image_resolve", image)

	if opts.ImageOverride == "" {
		opts.progress("image_building", "building "+image+" - first run takes minutes")
		if err := BuildArchitectImage(ctx, docker, opts.writer()); err != nil {
			return "", fmt.Errorf("ensure architect image: %w", err)
		}
	} else {
		exists, err := docker.ImageExists(ctx, image)
		if err != nil {
			return "", fmt.Errorf("checking image: %w", err)
		}
		if !exists {
			return "", fmt.Errorf("image %s not found", image)
		}
	}
	opts.progress("image_ready", image)

	opts.progress("credentials_sync", "syncing host credentials")
	_ = auth.SyncCredentials()

	envVars := []string{
		"SPWN_HOME=/home/spwn/.spwn",
	}

	architectLabels := map[string]string{labels.KindKey: labels.KindArchitect}
	labels.ApplyTestRun(architectLabels)
	containerCfg := &containerTypes.Config{
		Image:  image,
		Env:    envVars,
		Labels: architectLabels,
	}
	architectStackPath := platform.BaseDir() + "/architect/stack.md"
	if _, err := os.Stat(architectStackPath); os.IsNotExist(err) {
		_ = os.MkdirAll(platform.BaseDir()+"/architect", 0755)
		_ = os.WriteFile(architectStackPath, []byte("# Architect Stack\n\n## Focus\n\n## Queued\n- [ ] Review agent health and journal entries\n- [ ] Consolidate old agent memories\n\n## Done\n"), 0644)
	}

	opts.progress("host_files", "stack ready")

	hostCfg := &containerTypes.HostConfig{
		Binds: []string{
			platform.BaseDir() + ":/home/spwn/.spwn",
			architectStackPath + ":/me/stack.md",
			"/var/run/docker.sock:/var/run/docker.sock",
			platform.CredentialsDir() + ":/credentials:ro",
		},
		RestartPolicy: containerTypes.RestartPolicy{Name: "unless-stopped"},
	}

	opts.progress("container_creating", platform.ArchitectContainerName())
	id, err := docker.CreateNamedContainer(ctx, platform.ArchitectContainerName(), containerCfg, hostCfg)
	if err != nil {
		return "", fmt.Errorf("creating architect container: %w", err)
	}

	opts.progress("container_starting", id[:12])
	if err := docker.Start(ctx, id); err != nil {
		_ = docker.Remove(ctx, id)
		return "", fmt.Errorf("starting architect container: %w", err)
	}

	activity.Log(activity.Event{
		Type:   activity.TypeArchitectStarted,
		Actor:  "architect",
		Verb:   "started",
		Target: "architect",
		Phrase: activity.PhraseArchitectStarted(),
	})

	opts.progress("ready", id[:12])
	return id, nil
}

// StopDaemon stops and removes the spwn-architect container.
func StopDaemon(ctx context.Context) error {
	docker, err := backend.NewDocker()
	if err != nil {
		return fmt.Errorf("docker is not reachable: %w", err)
	}

	info, err := docker.Inspect(ctx, platform.ArchitectContainerName())
	if err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("architect is not running")
		}
		return fmt.Errorf("inspecting architect container: %w", err)
	}

	if info.Running {
		if err := docker.Stop(ctx, platform.ArchitectContainerName()); err != nil {
			return fmt.Errorf("stopping architect container: %w", err)
		}
	}

	if err := docker.Remove(ctx, platform.ArchitectContainerName()); err != nil {
		return fmt.Errorf("removing architect container: %w", err)
	}

	activity.Log(activity.Event{
		Type:   activity.TypeArchitectStopped,
		Actor:  "architect",
		Verb:   "stopped",
		Target: "architect",
		Phrase: activity.PhraseArchitectStopped(),
	})

	return nil
}

// GetDaemonStatus queries Docker for the architect container status.
func GetDaemonStatus(ctx context.Context) (*DaemonInfo, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("docker is not reachable: %w", err)
	}

	info, err := docker.Inspect(ctx, platform.ArchitectContainerName())
	if err != nil {
		if client.IsErrNotFound(err) {
			return &DaemonInfo{Running: false, Status: "not running"}, nil
		}
		return nil, fmt.Errorf("inspecting architect container: %w", err)
	}

	result := &DaemonInfo{
		ContainerID: info.ID[:12],
		Image:       info.Image,
		Status:      info.Status,
		Running:     info.Running,
		StartedAt:   info.StartedAt,
	}

	if info.Running {
		result.Uptime = time.Since(info.StartedAt)
	}

	return result, nil
}

// TalkExecArgs returns the docker exec arguments needed to talk to
// the Architect. The caller is responsible for executing the command
// (so it can handle interactive vs one-shot modes and streaming).
//
// If message is non-empty a one-shot --print invocation is returned;
// otherwise an interactive Claude session is returned.
func TalkExecArgs(message string) ([]string, error) {
	args := []string{"exec"}

	if message == "" {
		args = append(args, "-it")
	}

	args = append(args, "-u", "architect", "-w", "/me")
	args = append(args, "-e", "SPWN_HOME=/home/spwn/.spwn")

	_ = auth.SyncCredentials()

	args = append(args, platform.ArchitectContainerName())

	claudeArgs := []string{"claude", "--dangerously-skip-permissions"}
	if message != "" {
		claudeArgs = append(claudeArgs, "-p", message, "--print",
			"--append-system-prompt",
			"You are the Architect. Read /me/ARCHITECT.md for your identity. "+
				"IMPORTANT: When a user asks you to do something, you MUST include a [STACK_PUSH] marker in your response. "+
				"Format: [STACK_PUSH] Short task title\\nPriority: blocking|queued\\nBrief description. "+
				"Also update /me/stack.md with the new task. "+
				"When completing a task use [STACK_POP] Short task title. "+
				"Read /me/skills/ for detailed guides.")
	}

	escaped := make([]string, len(claudeArgs))
	for i, arg := range claudeArgs {
		escaped[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	setup := "source /credentials/.env 2>/dev/null"
	setup += "; [ -f /credentials/openai/auth.json ] && mkdir -p $HOME/.codex && ln -sf /credentials/openai/auth.json $HOME/.codex/auth.json 2>/dev/null"
	shellCmd := setup + "; exec " + strings.Join(escaped, " ")
	args = append(args, "bash", "-c", shellCmd)
	return args, nil
}
