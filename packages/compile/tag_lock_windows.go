//go:build windows

package compile

import "context"

func lockTag(context.Context, string) (func(), error) {
	return func() {}, nil
}

func holdTag(context.Context, string) (func(), error) {
	return func() {}, nil
}

func claimTag(string) (func(), bool, error) {
	return nil, false, nil
}
