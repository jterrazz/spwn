//go:build windows

package compile

import "context"

func lockTag(context.Context, string) (func(), error) {
	return func() {}, nil
}
