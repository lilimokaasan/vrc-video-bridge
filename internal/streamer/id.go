package streamer

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func newJobID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}
	return time.Now().UTC().Format("20060102150405") + "-" + hex.EncodeToString(buf[:])
}
