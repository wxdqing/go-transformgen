// Package msgid computes deterministic protocol message ids.
//
// The scheme mirrors the TianLong3 client generator so that transformgen can be
// the single source of truth for both the Go server and the C# client:
//
//	id = |netHash(name) % 90000000| + band
//	band = 200000000 for client->server (request) messages
//	band = 100000000 for server->client (response/notify) messages
//
// netHash reproduces the .NET Framework deterministic string hash. The algorithm
// is fixed on purpose: identical message names always map to identical ids, so
// independently generated Go and C# code interoperate without any shared table.
package msgid

import "unicode/utf16"

const (
	bandModulus     int32 = 90000000
	toClientBandMin int32 = 100000000
	toServerBandMin int32 = 200000000

	// hashMultiplier and hashSeed are the constants used by the .NET Framework
	// non-randomized String.GetHashCode implementation.
	hashSeed       int32 = 5381
	hashMultiplier int32 = 1566083941
)

// netHash reproduces the .NET Framework deterministic string hash over UTF-16
// code units. Go's fixed-width int32 arithmetic wraps on overflow, matching the
// unchecked arithmetic of the original implementation.
func netHash(s string) int32 {
	units := utf16.Encode([]rune(s))
	h1 := hashSeed
	h2 := hashSeed
	for i := 0; i < len(units); i += 2 {
		h1 = ((h1 << 5) + h1) ^ int32(units[i])
		if i+1 < len(units) {
			h2 = ((h2 << 5) + h2) ^ int32(units[i+1])
		}
	}
	return h1 + h2*hashMultiplier
}

// Compute returns the deterministic message id for name in the given direction.
// toServer selects the client->server band (request messages); everything else
// uses the server->client band (response and notify messages).
func Compute(name string, toServer bool) uint32 {
	m := netHash(name) % bandModulus
	if m < 0 {
		m += bandModulus
	}
	band := toClientBandMin
	if toServer {
		band = toServerBandMin
	}
	return uint32(m + band)
}
