package idgen

import (
	"github.com/btcsuite/btcutil/base58"
)

func NextSID() string {
	id := NewObjectID()
	return base58.Encode(id[:])
}
