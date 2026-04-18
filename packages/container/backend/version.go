package backend

// NeedsRebuild reports whether an image should be rebuilt based on its current
// version label versus the expected version. It returns true when the image is
// missing (currentVersion is empty) or when the versions do not match.
func NeedsRebuild(currentVersion, expectedVersion string) bool {
	return currentVersion == "" || currentVersion != expectedVersion
}
