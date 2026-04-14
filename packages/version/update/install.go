package update

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PlatformToken returns the GoReleaser asset name token for the current
// runtime - e.g. "darwin_arm64", "linux_amd64", "windows_amd64".
func PlatformToken() string {
	return fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
}

// ExtractBinary opens a .tar.gz archive, finds the first regular file
// whose name is `binaryName` (possibly nested in a directory), and writes
// it to destPath with executable permissions. Returns an error if no
// matching file is found.
func ExtractBinary(archivePath, binaryName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("binary %q not found in archive %s", binaryName, archivePath)
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != binaryName {
			continue
		}
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("create dest: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			os.Remove(destPath)
			return fmt.Errorf("copy: %w", err)
		}
		if err := out.Close(); err != nil {
			return err
		}
		return nil
	}
}

// AtomicReplace overwrites `target` with the contents of `newFile` as
// atomically as the OS allows:
//
//   - POSIX: os.Rename into place - the kernel guarantees readers see either
//     the old or the new binary, never a partial write.
//   - Windows: os.Rename fails when the target is running, so we fall back
//     to renaming the old binary aside (`target.old`) and then renaming the
//     new one into place, scheduling the old one for deletion.
//
// The target directory must be writable by the current user; if it's not
// (e.g. /usr/local/bin owned by root), the caller should re-invoke itself
// under sudo or ask the user to install to a writable prefix.
func AtomicReplace(newFile, target string) error {
	// Ensure the new binary is executable.
	if err := os.Chmod(newFile, 0755); err != nil {
		return fmt.Errorf("chmod new binary: %w", err)
	}

	if runtime.GOOS == "windows" {
		return atomicReplaceWindows(newFile, target)
	}

	// POSIX: same filesystem → atomic rename. The OS frees the inode of
	// the running binary when the last process holding it exits.
	if err := os.Rename(newFile, target); err != nil {
		return fmt.Errorf("rename %s → %s: %w", newFile, target, err)
	}
	return nil
}

func atomicReplaceWindows(newFile, target string) error {
	// Windows holds a lock on the running .exe; move the old aside first.
	old := target + ".old"
	_ = os.Remove(old) // stale from previous failed upgrade
	if err := os.Rename(target, old); err != nil {
		return fmt.Errorf("move old binary aside: %w", err)
	}
	if err := os.Rename(newFile, target); err != nil {
		// Try to roll back.
		_ = os.Rename(old, target)
		return fmt.Errorf("rename new binary: %w", err)
	}
	// `old` will be removed at next launch / by the OS eventually.
	return nil
}

// IsWritable reports whether the current process can replace the file at
// path (or, if it doesn't exist, create a file in that directory).
func IsWritable(path string) bool {
	dir := filepath.Dir(path)
	test := filepath.Join(dir, ".spwn-write-test")
	f, err := os.Create(test)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(test)
	return true
}

// AssetNameFor returns the expected GoReleaser archive name for the given
// binary name + platform. E.g. AssetNameFor("spwn", "darwin_arm64") →
// "spwn_darwin_arm64.tar.gz".
func AssetNameFor(binary, platform string) string {
	ext := ".tar.gz"
	if strings.HasPrefix(platform, "windows_") {
		ext = ".zip"
	}
	return fmt.Sprintf("%s_%s%s", binary, platform, ext)
}
