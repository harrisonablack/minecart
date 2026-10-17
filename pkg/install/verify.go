package install

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"os"
	"sort"

	"github.com/harrisonablack/minecart/pkg/artifact"
)

// hashPriority is the order in which hash algorithms are preferred.
var hashPriority = []string{"sha256", "sha512", "sha1", "md5"}

// hashFuncs maps an algorithm name to its constructor.
var hashFuncs = map[string]func() hash.Hash{
	"sha256": sha256.New,
	"sha512": sha512.New,
	"sha1":   sha1.New,
	"md5":    md5.New,
}

// strongestHash returns the strongest supported algorithm and its digest from
// the artifact's hashes, or ("", "") if none is supported.
func strongestHash(hashes map[string]string) (string, string) {
	for _, algo := range hashPriority {
		if digest, ok := hashes[algo]; ok {
			return algo, digest
		}
	}
	return "", ""
}

// hashAlgorithms returns the sorted algorithm names present in hashes.
func hashAlgorithms(hashes map[string]string) []string {
	algos := make([]string, 0, len(hashes))
	for algo := range hashes {
		algos = append(algos, algo)
	}
	sort.Strings(algos)
	return algos
}

// matches reports whether the file at path matches the artifact's strongest
// supported hash. It returns false when there is no hash to check or the file
// cannot be read.
func matches(a artifact.Artifact, path string) bool {
	algo, digest := strongestHash(a.Hashes)
	if algo == "" {
		return false
	}
	actual, err := hashFile(path, algo)
	if err != nil {
		return false
	}
	return actual == digest
}

// verify checks the file at path against the artifact's hashes, logging the
// outcome. It returns an error only on mismatch.
func verify(a artifact.Artifact, path string, logger *slog.Logger) error {
	algo, digest := strongestHash(a.Hashes)

	if algo == "" {
		if len(a.Hashes) == 0 {
			logger.Warn("no hash available, skipping verification", "artifact", a.Name)
		} else {
			logger.Warn("no supported hash, skipping verification",
				"artifact", a.Name, "returned", hashAlgorithms(a.Hashes))
		}
		return nil
	}

	actual, err := hashFile(path, algo)
	if err != nil {
		return fmt.Errorf("hash %s: %w", algo, err)
	}

	if actual != digest {
		logger.Error("hash mismatch", "artifact", a.Name, "algorithm", algo, "want", digest, "got", actual)
		return fmt.Errorf("%s: %s hash mismatch", a.Name, algo)
	}

	logger.Info("verified", "artifact", a.Name, "algorithm", algo)
	return nil
}

// hashFile computes the hash of the file at path using the given algorithm.
func hashFile(path, algo string) (string, error) {
	newHash, ok := hashFuncs[algo]
	if !ok {
		return "", fmt.Errorf("unsupported hash algorithm %q", algo)
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := newHash()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
