package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAllWithAllTrue tests All when all elements satisfy the predicate
func TestAllWithAllTrue(t *testing.T) {
	numbers := []int{2, 4, 6, 8}
	result := All(numbers, func(n int) bool { return n%2 == 0 })
	if !result {
		t.Errorf("All() = %v, want true", result)
	}
}

// TestAllWithSomeFalse tests All when some elements don't satisfy the predicate
func TestAllWithSomeFalse(t *testing.T) {
	numbers := []int{2, 4, 5, 8}
	result := All(numbers, func(n int) bool { return n%2 == 0 })
	if result {
		t.Errorf("All() = %v, want false", result)
	}
}

// TestAllWithEmptySlice tests All with an empty slice
func TestAllWithEmptySlice(t *testing.T) {
	numbers := []int{}
	result := All(numbers, func(n int) bool { return n%2 == 0 })
	if !result {
		t.Errorf("All() with empty slice = %v, want true", result)
	}
}

// TestAllWithSingleTrue tests All with a single element that satisfies the predicate
func TestAllWithSingleTrue(t *testing.T) {
	numbers := []int{2}
	result := All(numbers, func(n int) bool { return n%2 == 0 })
	if !result {
		t.Errorf("All() = %v, want true", result)
	}
}

// TestAllWithSingleFalse tests All with a single element that doesn't satisfy the predicate
func TestAllWithSingleFalse(t *testing.T) {
	numbers := []int{3}
	result := All(numbers, func(n int) bool { return n%2 == 0 })
	if result {
		t.Errorf("All() = %v, want false", result)
	}
}

// TestAllWithStrings tests All with string elements
func TestAllWithStrings(t *testing.T) {
	words := []string{"hello", "world", "go"}
	result := All(words, func(s string) bool { return len(s) > 0 })
	if !result {
		t.Errorf("All() with strings = %v, want true", result)
	}
}

// TestSomeWithAtLeastOneTrue tests Some when at least one element satisfies the predicate
func TestSomeWithAtLeastOneTrue(t *testing.T) {
	numbers := []int{1, 2, 3, 4}
	result := Some(numbers, func(n int) bool { return n%2 == 0 })
	if !result {
		t.Errorf("Some() = %v, want true", result)
	}
}

// TestSomeWithAllTrue tests Some when all elements satisfy the predicate
func TestSomeWithAllTrue(t *testing.T) {
	numbers := []int{2, 4, 6, 8}
	result := Some(numbers, func(n int) bool { return n%2 == 0 })
	if !result {
		t.Errorf("Some() = %v, want true", result)
	}
}

// TestSomeWithNoneSatisfy tests Some when no elements satisfy the predicate
func TestSomeWithNoneSatisfy(t *testing.T) {
	numbers := []int{1, 3, 5, 7}
	result := Some(numbers, func(n int) bool { return n%2 == 0 })
	if result {
		t.Errorf("Some() = %v, want false", result)
	}
}

// TestSomeWithEmptySlice tests Some with an empty slice
func TestSomeWithEmptySlice(t *testing.T) {
	numbers := []int{}
	result := Some(numbers, func(n int) bool { return n%2 == 0 })
	if result {
		t.Errorf("Some() with empty slice = %v, want false", result)
	}
}

// TestSomeWithSingleTrue tests Some with a single element that satisfies the predicate
func TestSomeWithSingleTrue(t *testing.T) {
	numbers := []int{2}
	result := Some(numbers, func(n int) bool { return n%2 == 0 })
	if !result {
		t.Errorf("Some() = %v, want true", result)
	}
}

// TestSomeWithSingleFalse tests Some with a single element that doesn't satisfy the predicate
func TestSomeWithSingleFalse(t *testing.T) {
	numbers := []int{3}
	result := Some(numbers, func(n int) bool { return n%2 == 0 })
	if result {
		t.Errorf("Some() = %v, want false", result)
	}
}

// TestMapWithIntegers tests Map with integer transformation
func TestMapWithIntegers(t *testing.T) {
	numbers := []int{1, 2, 3, 4}
	result := Map(numbers, func(n int) int { return n * 2 })
	expected := []int{2, 4, 6, 8}

	if len(result) != len(expected) {
		t.Errorf("Map() length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Map() result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestMapWithIntToString tests Map with type conversion
func TestMapWithIntToString(t *testing.T) {
	numbers := []int{1, 2, 3}
	result := Map(numbers, func(n int) string { return string(rune('0' + n)) })
	expected := []string{"1", "2", "3"}

	if len(result) != len(expected) {
		t.Errorf("Map() length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Map() result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestMapWithEmptySlice tests Map with an empty slice
func TestMapWithEmptySlice(t *testing.T) {
	numbers := []int{}
	result := Map(numbers, func(n int) int { return n * 2 })

	if len(result) != 0 {
		t.Errorf("Map() with empty slice length = %d, want 0", len(result))
	}
}

// TestMapWithStrings tests Map with string transformation
func TestMapWithStrings(t *testing.T) {
	words := []string{"hello", "world"}
	result := Map(words, func(s string) int { return len(s) })
	expected := []int{5, 5}

	if len(result) != len(expected) {
		t.Errorf("Map() length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Map() result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestFlatMapWithIntSlices tests FlatMap with integer slices
func TestFlatMapWithIntSlices(t *testing.T) {
	numbers := []int{1, 2, 3}
	result := FlatMap(numbers, func(n int) []int {
		return []int{n, n * 2}
	})
	expected := []int{1, 2, 2, 4, 3, 6}

	if len(result) != len(expected) {
		t.Errorf("FlatMap() length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("FlatMap() result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestFlatMapWithEmptySlice tests FlatMap with an empty slice
func TestFlatMapWithEmptySlice(t *testing.T) {
	numbers := []int{}
	result := FlatMap(numbers, func(n int) []int {
		return []int{n, n * 2}
	})

	if len(result) != 0 {
		t.Errorf("FlatMap() with empty slice length = %d, want 0", len(result))
	}
}

// TestFlatMapWithStringToChars tests FlatMap converting strings to character slices
func TestFlatMapWithStringToChars(t *testing.T) {
	words := []string{"hi", "go"}
	result := FlatMap(words, func(s string) []rune {
		return []rune(s)
	})
	expected := []rune{'h', 'i', 'g', 'o'}

	if len(result) != len(expected) {
		t.Errorf("FlatMap() length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("FlatMap() result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestFlatMapWithVariableSizeSlices tests FlatMap with variable-size inner slices
func TestFlatMapWithVariableSizeSlices(t *testing.T) {
	numbers := []int{1, 2, 3}
	result := FlatMap(numbers, func(n int) []int {
		result := make([]int, n)
		for i := 0; i < n; i++ {
			result[i] = n
		}
		return result
	})
	expected := []int{1, 2, 2, 3, 3, 3}

	if len(result) != len(expected) {
		t.Errorf("FlatMap() length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("FlatMap() result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

func TestRetryImmediateSuccess(t *testing.T) {
	calls := 0
	if err := Retry(func() error {
		calls++
		return nil
	}, time.Millisecond, 10*time.Millisecond); err != nil {
		t.Fatalf("Retry() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Errorf("Retry() calls = %d, want 1", calls)
	}
}

func TestRetryTransientFailureThenSuccess(t *testing.T) {
	wantErr := errors.New("transient")
	calls := 0
	if err := Retry(func() error {
		calls++
		if calls < 3 {
			return wantErr
		}
		return nil
	}, time.Millisecond, 50*time.Millisecond); err != nil {
		t.Fatalf("Retry() error = %v, want nil", err)
	}
	if calls != 3 {
		t.Errorf("Retry() calls = %d, want 3", calls)
	}
}

func TestRetryAlwaysFailureReturnsLastErrorWithinDuration(t *testing.T) {
	lastErr := errors.New("last")
	calls := 0
	start := time.Now()
	err := Retry(func() error {
		calls++
		return lastErr
	}, time.Millisecond, 5*time.Millisecond)
	elapsed := time.Since(start)

	if !errors.Is(err, lastErr) {
		t.Errorf("Retry() error = %v, want %v", err, lastErr)
	}
	if calls < 2 {
		t.Errorf("Retry() calls = %d, want retries", calls)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("Retry() elapsed = %v, exceeded max duration", elapsed)
	}
}

func TestVerifySHA256(t *testing.T) {
	// sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	const helloHash = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	t.Run("match", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "data")
		if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
		if !VerifySHA256(helloHash, tmpFile) {
			t.Error("expected verification to succeed")
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "data")
		if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
		if VerifySHA256(strings.Repeat("0", 64), tmpFile) {
			t.Error("expected verification to fail")
		}
	})

	t.Run("case-insensitive", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "data")
		if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
		if !VerifySHA256(strings.ToUpper(helloHash), tmpFile) {
			t.Error("expected verification to succeed with uppercase digest")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if VerifySHA256(helloHash, filepath.Join(t.TempDir(), "nope")) {
			t.Error("expected verification to fail for missing file")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "data")
		if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
		if VerifySHA256("", tmpFile) {
			t.Error("expected verification to fail with empty body")
		}
	})
}
