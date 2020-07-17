package fileset

import (
	"io"

	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/index"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/tar"
)

type FileReaderAPI interface {
	Index() *index.Index
	Header() (*tar.Header, error)
	Get(w io.Writer) error
	GetContents(w io.Writer) error
}

var _ FileReaderAPI = &FileMergeReader{}
var _ FileReaderAPI = &FileReader{}

type ReaderAPI interface {
	Iterate(cb func(FileReaderAPI) error, stopBefore ...string) error
}

var _ ReaderAPI = &MergeReader{}
var _ ReaderAPI = &Reader{}
