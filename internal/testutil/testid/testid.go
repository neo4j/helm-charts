// Package testid derives deterministic, parallel-safe identifiers for tests
// that must allocate named resources in a shared environment (Kubernetes
// namespaces, cloud storage buckets, etc.). Its job is to guarantee that two
// tests running in parallel — even in the same binary — never produce the
// same identifier, without forcing callers to invent ad-hoc suffixes.
package testid

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

// For returns a short, deterministic identifier derived from runID and t.Name().
// The result contains only characters safe for Kubernetes resource names
// (lowercase alphanumerics and '-'), and is bounded in length (runID +
// 7 characters) so callers can append domain-specific prefixes without
// blowing the 63-character DNS-1123 limit.
//
// Two tests with different t.Name() values always get different identifiers;
// the same test always gets the same identifier across calls within one run.
func For(runID string, t testing.TB) string {
	h := sha1.Sum([]byte(t.Name()))
	return runID + "-" + hex.EncodeToString(h[:])[:6]
}
