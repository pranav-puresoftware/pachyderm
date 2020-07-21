package fileset

import (
	"io"

	"github.com/pachyderm/pachyderm/src/client/pkg/errors"
	"github.com/pachyderm/pachyderm/src/server/pkg/obj"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/chunk"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/tar"
)

// WithLocalStorage constructs a local storage instance for testing during the lifetime of
// the callback.
func WithLocalStorage(f func(*Storage) error) error {
	return chunk.WithLocalStorage(func(objC obj.Client, chunks *chunk.Storage) error {
		return f(NewStorage(objC, chunks))
	})
}

func CopyFiles(w *Writer, r FileSource) error {
	switch r := r.(type) {
	case *Reader:
		return r.iterate(func(fr *FileReader) error {
			return w.CopyFile(fr)
		})
	default:
		return errors.Errorf("CopyFiles does not support reader type: %T", r)
	}
}

func WriteTarEntry(w io.Writer, f File) error {
	h, err := f.Header()
	if err != nil {
		return err
	}
	tw := tar.NewWriter(w)
	if err := tw.WriteHeader(h); err != nil {
		return err
	}
	return f.Content(tw)
}
