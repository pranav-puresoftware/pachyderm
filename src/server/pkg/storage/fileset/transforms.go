package fileset

import (
	"io"

	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/index"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/tar"
)

var _ FileSource = &HeaderFilter{}

type HeaderFilter struct {
	F func(th *tar.Header) bool
	R FileSource
}

func (hf *HeaderFilter) Iterate(cb func(File) error, stopBefore ...string) error {
	return hf.R.Iterate(func(fr File) error {
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

type IndexFilter struct {
	F func(idx *index.Index) bool
	R FileSource
}

func (fil *IndexFilter) Iterate(cb func(File) error, stopBefore ...string) error {
	return fil.R.Iterate(func(fr File) error {
		idx := fr.Index()
		if fil.F(idx) {
			cb(fr)
		}
		return nil
	})
}

var _ FileSource = &HeaderMapper{}

type HeaderMapper struct {
	F func(th *tar.Header) *tar.Header
	R FileSource
}

func (hm *HeaderMapper) Iterate(cb func(File) error, stopBefore ...string) error {
	return hm.R.Iterate(func(fr File) error {
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

var _ File = headerMap{}

type headerMap struct {
	header *tar.Header
	inner  File
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
	if err := hm.inner.Content(w); err != nil {
		return err
	}
	return tw.Flush()
}

func (hm headerMap) Content(w io.Writer) error {
	return hm.inner.Content(w)
}
