package fileset

import (
	"github.com/pachyderm/pachyderm/src/client/pkg/errors"
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

func CopyFiles(w *Writer, r ReaderAPI) error {
	switch r := r.(type) {
	case *Reader:
		return r.iterate(func(fr *FileReader) error {
			return w.CopyFile(fr)
		})
	default:
		return errors.Errorf("CopyFiles does not support reader type: %T", r)
	}
}
