package id

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/jterrazz/universe/wordlist"
)

// Generate returns a human-readable universe ID in the format u-{adj}-{noun}-{5digits}.
func Generate() string {
	adj := pick(wordlist.Adjectives)
	noun := pick(wordlist.Nouns)
	suffix := randDigits(5)
	return fmt.Sprintf("u-%s-%s-%s", adj, noun, suffix)
}

func pick(list []string) string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(list))))
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return list[n.Int64()]
}

func randDigits(count int) string {
	digits := make([]byte, count)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		digits[i] = byte('0') + byte(n.Int64())
	}
	return string(digits)
}
