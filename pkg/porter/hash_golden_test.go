package porter

import "testing"

// TestFoldDataHash_GoldenCJK pins the Data fold recipe against a cross-repo
// golden digest. The expected value is the one it-claws' tests/test_hashes.py
// pins (comment there: cross-repo golden). The CJK arcname exercises the
// byte-length prefix for multi-byte UTF-8 names. If this fails, the fold
// recipe has drifted and the change must be reported, not silently fixed.
func TestFoldDataHash_GoldenCJK(t *testing.T) {
	got := foldDataHash(map[string]string{
		"conf/data.db":        "0e6612c33396e3c5732b649d8182c81d08f98a96",
		"drivers/中文/файл.bin": "6238b1f42fbbcd8725cb23928608bcbf22dc593b",
		"drivers/gpu/amd.bin": "397520b0bf01806934b3b99eacd1aca2b3c44930",
	})
	const want = "cb58163e3c641021b923f1268ff49f09eb8082e5"
	if got != want {
		t.Fatalf("foldDataHash golden mismatch: got %q want %q", got, want)
	}
}
