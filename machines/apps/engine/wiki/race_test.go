//go:build race

package wiki_test

// value: how much slower the race detector runs a path, so a time budget
// holds under -race.
const slowdown = 10
