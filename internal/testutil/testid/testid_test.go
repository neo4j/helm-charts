package testid

import (
	"regexp"
	"sync"
	"testing"
)

func TestForIsStableAndUnique(t *testing.T) {
	t.Parallel()

	const runID = "runabc"
	first := For(runID, t)
	second := For(runID, t)
	if first != second {
		t.Fatalf("For must be stable across calls for the same t; got %q then %q", first, second)
	}

	dns := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !dns.MatchString(first) {
		t.Fatalf("result %q must match DNS-1123-safe character set", first)
	}

	// Names below would have collided under the old "<prefix>-<runID>" scheme
	// when two parallel tests happened to share the same prefix.
	names := []string{
		"AuthInvalidPassword_case1",
		"AuthInvalidPassword_case2",
		"LdapAuthWrongKey_case1",
		"LdapAuthWrongKey_case2",
		"ParallelA",
		"ParallelB",
	}

	var mu sync.Mutex
	seen := make(map[string]string, len(names))
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			id := For(runID, t)
			mu.Lock()
			defer mu.Unlock()
			if prior, clash := seen[id]; clash {
				t.Fatalf("collision: %s and %s both produced %q", t.Name(), prior, id)
			}
			seen[id] = t.Name()
		})
	}
}

func TestForVariesWithRunID(t *testing.T) {
	t.Parallel()
	a := For("alpha", t)
	b := For("beta", t)
	if a == b {
		t.Fatalf("identifiers must differ across runIDs; got %q for both", a)
	}
}
