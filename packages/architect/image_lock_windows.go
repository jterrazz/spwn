//go:build windows

package architect

import "context"

func lockWorldImage(context.Context) (func(), error) {
	return func() {}, nil
}
