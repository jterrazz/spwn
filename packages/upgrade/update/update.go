package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Plan is the result of CheckForUpdate: everything the caller needs to
// decide whether to proceed and, if so, to download+install the update.
type Plan struct {
	Current       Version
	Latest        Version
	Release       *Release
	Asset         *Asset   // platform-matched binary archive
	ChecksumAsset *Asset   // SHA256SUMS file in the same release (optional)
	Platform      string   // GOOS_GOARCH
	UpdateAvail   bool     // true when Latest > Current (and Current isn't same as Latest)
	Notes         []string // human-readable lines to display before installing
}

// CheckOpts configures CheckForUpdate.
type CheckOpts struct {
	Channel  Channel
	Platform string // override, defaults to runtime GOOS_GOARCH
}

// CheckForUpdate fetches the latest release on the requested channel and
// decides whether an update is available for the given platform. Does not
// download anything - the caller should inspect plan.UpdateAvail and then
// call Apply() if it wants to proceed.
func CheckForUpdate(ctx context.Context, client ReleaseClient, currentVersion string, opts CheckOpts) (*Plan, error) {
	cur, err := ParseVersion(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("parse current version %q: %w", currentVersion, err)
	}

	release, err := ResolveTarget(ctx, client, opts.Channel)
	if err != nil {
		return nil, err
	}

	latest, err := ParseVersion(release.TagName)
	if err != nil {
		return nil, fmt.Errorf("parse release tag %q: %w", release.TagName, err)
	}

	platform := opts.Platform
	if platform == "" {
		platform = PlatformToken()
	}

	plan := &Plan{
		Current:       cur,
		Latest:        latest,
		Release:       release,
		Asset:         FindAsset(release, platform),
		ChecksumAsset: FindChecksumsAsset(release),
		Platform:      platform,
		UpdateAvail:   cur.Compare(latest) < 0,
	}
	return plan, nil
}

// ApplyOpts configures Apply.
type ApplyOpts struct {
	BinaryName string // name of the binary inside the tar.gz (e.g. "spwn")
	TargetPath string // where the current binary lives on disk (os.Executable())
	WorkDir    string // temp scratch dir; if empty, uses os.MkdirTemp
	// Progress is called with human-readable status lines so the CLI can
	// render them with its stepper. Safe to leave nil.
	Progress func(msg string)
}

// Apply executes the plan: downloads the asset, verifies the checksum
// against the checksums file (if present), extracts the binary, and
// atomically replaces TargetPath.
func Apply(ctx context.Context, plan *Plan, opts ApplyOpts) error {
	if plan == nil || plan.Asset == nil {
		return fmt.Errorf("no asset found for platform %s in release %s", plan.Platform, plan.Release.TagName)
	}
	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}

	workDir := opts.WorkDir
	if workDir == "" {
		d, err := os.MkdirTemp("", "spwn-upgrade-")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
		defer os.RemoveAll(d)
		workDir = d
	}

	// 1) Download the archive.
	progress(fmt.Sprintf("Downloading %s", plan.Asset.Name))
	archivePath := filepath.Join(workDir, plan.Asset.Name)
	if _, err := DownloadHTTP(ctx, plan.Asset.DownloadURL, archivePath); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}

	// 2) Verify checksum (skip silently if the release doesn't ship one).
	if plan.ChecksumAsset != nil {
		progress("Verifying checksum")
		checksumsPath := filepath.Join(workDir, plan.ChecksumAsset.Name)
		if _, err := DownloadHTTP(ctx, plan.ChecksumAsset.DownloadURL, checksumsPath); err != nil {
			return fmt.Errorf("download checksums: %w", err)
		}
		f, err := os.Open(checksumsPath)
		if err != nil {
			return err
		}
		checksums, err := ParseChecksums(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("parse checksums: %w", err)
		}
		digest, ok := checksums[plan.Asset.Name]
		if !ok {
			return fmt.Errorf("no checksum entry for %s in checksums.txt - refusing to install", plan.Asset.Name)
		}
		if err := VerifyChecksum(archivePath, digest); err != nil {
			return err
		}
	}

	// 3) Extract the binary.
	progress("Extracting binary")
	newBin := filepath.Join(workDir, "spwn.new")
	binaryName := opts.BinaryName
	if binaryName == "" {
		binaryName = "spwn"
	}
	if strings.HasSuffix(plan.Asset.Name, ".zip") {
		// .zip handling is Windows-only; implement when we ship a Windows build.
		return fmt.Errorf("zip archives not yet supported; open an issue if you need Windows support")
	}
	if err := ExtractBinary(archivePath, binaryName, newBin); err != nil {
		return err
	}

	// 4) Atomic replace.
	progress("Installing")
	if !IsWritable(opts.TargetPath) {
		return fmt.Errorf("cannot write to %s\nRun with elevated privileges or reinstall to a user-writable path", opts.TargetPath)
	}
	return AtomicReplace(newBin, opts.TargetPath)
}
