package fileset

import (
	"strings"

	"github.com/pachyderm/pachyderm/src/server/pkg/obj"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/chunk"
)

// WithLocalStorage constructs a local storage instance for testing during the lifetime of
// the callback.
func WithLocalStorage(f func(*Storage) error) error {
	return chunk.WithLocalStorage(func(objC obj.Client, chunks *chunk.Storage) error {
		return f(NewStorage(objC, chunks))
	})
}

// CleanTarPath ensures that the path is in the canonical format for tar header names.
// This includes removing prepending /'s and ensure directory paths
// have a trailing slash.
func CleanTarPath(x string, isDir bool) string {
	y := strings.Trim(x, "/")
	if isDir {
		y += "/"
	}
	return y
}

// IsCleanTarPath determines if the path is a valid tar path.
func IsCleanTarPath(x string, isDir bool) bool {
	y := CleanTarPath(x, isDir)
	return y == x
}
