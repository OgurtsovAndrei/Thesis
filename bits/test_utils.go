package bits

import (
	"math/rand"
	"strings"
	"time"
)

// randomTextString generates a random ASCII text string for CharBitString
func randomTextString(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		// Generate printable ASCII characters (32-126)
		char := byte(32 + r.Intn(95))
		sb.WriteByte(char)
	}
	return sb.String()
}

// randomUint64 generates a random uint64 value for Uint64BitString
func randomUint64() uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Uint64()
}

// randomBinaryString generates a random string of '0's and '1's
func randomBinaryString(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		if r.Intn(2) == 0 {
			sb.WriteByte('0')
		} else {
			sb.WriteByte('1')
		}
	}
	return sb.String()
}

// randomBase64String generates a random base64 string for trie.BitString
func randomBase64String(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(base64Chars[r.Intn(len(base64Chars))])
	}
	return sb.String()
}
