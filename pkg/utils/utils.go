package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Returns true if all elements in the slice satisfy the predicate.
func All[T any](ts []T, pred func(T) bool) bool {
	for _, t := range ts {
		if !pred(t) {
			return false
		}
	}
	return true
}

// Returns true if at least one element in the slice satisfies the predicate.
func Some[T any](ts []T, pred func(T) bool) bool {
	for _, t := range ts {
		if pred(t) {
			return true
		}
	}
	return false
}

func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func FlatMap[A, B any](input []A, f func(A) []B) []B {
	var result []B
	for _, v := range input {
		result = append(result, f(v)...)
	}
	return result
}

// Retry calls fn immediately and retries failed calls until maxDuration elapses.
// It returns nil on success or the last error returned by fn.
func Retry(fn func() error, interval, maxDuration time.Duration) error {
	deadline := time.Now().Add(maxDuration)
	err := fn()
	for err != nil {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return err
		}
		if interval > remaining {
			interval = remaining
		}
		time.Sleep(interval)
		if !time.Now().Before(deadline) {
			return err
		}
		err = fn()
	}
	return nil
}

// Returns true iff the SHA-256 of filePath matches the digest in body.
func VerifySHA256(expected, filePath string) (ok bool) {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return strings.EqualFold(expected, fmt.Sprintf("%x", h.Sum(nil)))
}
