package fileset

import (
	"io"

	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/index"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/tar"
)

var _ ReaderAPI = &HeaderFilter{}

type HeaderFilter struct {
	F func(th *tar.Header) bool
	R ReaderAPI
}

func (hf *HeaderFilter) Iterate(cb func(FileReaderAPI) error, stopBefore ...string) error {
	return hf.R.Iterate(func(fr FileReaderAPI) error {
		th, err := fr.Header()
		if err != nil {
			return err
		}
		if hf.F(th) {
			return cb(fr)
		}
		return nil
	}, stopBefore...)
}

var _ ReaderAPI = &HeaderMapper{}

type HeaderMapper struct {
	F func(th *tar.Header) *tar.Header
	R ReaderAPI
}

func (hm *HeaderMapper) Iterate(cb func(FileReaderAPI) error, stopBefore ...string) error {
	return hm.R.Iterate(func(fr FileReaderAPI) error {
		x, err := fr.Header()
		if err != nil {
			return err
		}
		y := hm.F(x)
		return cb(headerMap{
			header: y,
			inner:  fr,
		})
	})
}

var _ FileReaderAPI = headerMap{}

type headerMap struct {
	header *tar.Header
	inner  FileReaderAPI
}

func (hm headerMap) Index() *index.Index {
	// TODO: should be able to transform the path?
	return hm.Index()
}

func (hm headerMap) Header() (*tar.Header, error) {
	return hm.header, nil
}

func (hm headerMap) Get(w io.Writer) error {
	tw := tar.NewWriter(w)
	if err := tw.WriteHeader(hm.header); err != nil {
		return err
	}
	if err := hm.inner.GetContents(w); err != nil {
		return err
	}
	return tw.Flush()
}

func (hm headerMap) GetContents(w io.Writer) error {
	return hm.inner.GetContents(w)
}
