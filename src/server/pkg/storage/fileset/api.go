package fileset

import (
	"io"

	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/index"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/tar"
)

type File interface {
	Index() *index.Index
	Header() (*tar.Header, error)
	Content(w io.Writer) error
}

var _ File = &FileMergeReader{}
var _ File = &FileReader{}

type FileSource interface {
	Iterate(cb func(File) error, stopBefore ...string) error
}

var _ FileSource = &MergeReader{}
var _ FileSource = &Reader{}
