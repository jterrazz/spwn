package kit

import "github.com/jterrazz/studio-station/apps/engine/core/secret"

func Typed(answer string) secret.Sent {
	// value: a secret the owner typed into a hidden field, held as a request
	// holds one, so it prints as its marker from here on.
	return secret.Sent{Value: secret.Hold("", []byte(answer))}
}
