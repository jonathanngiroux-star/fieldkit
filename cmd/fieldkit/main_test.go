package main

import (
	"bytes"
	"strings"
	"testing"
)

// version prints the version string in the expected shape.
func TestVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"version"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d, stderr %s", code, errb.String())
	}
	if !strings.HasPrefix(out.String(), "fieldkit ") {
		t.Fatalf("version output shape: %q", out.String())
	}
}

// donate must print both addresses exactly, each exactly once, and no
// payment-ask language.
func TestDonateExact(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"donate"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got := out.String()
	const eth = "0x85ee7E71f762d772599cbF1EC20E651B30657521"
	const btc = "bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg"
	if strings.Count(got, eth) != 1 {
		t.Errorf("eth address count = %d, want 1", strings.Count(got, eth))
	}
	if strings.Count(got, btc) != 1 {
		t.Errorf("btc address count = %d, want 1", strings.Count(got, btc))
	}
	for _, banned := range []string{"stripe", "invoice", "pricing", "upgrade", "billing", "contact sales", "subscription"} {
		if strings.Contains(strings.ToLower(got), banned) {
			t.Errorf("donate output contains %q", banned)
		}
	}
}
