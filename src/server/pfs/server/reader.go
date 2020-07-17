package server

import (
	"hash"
	"io"

	"github.com/pachyderm/pachyderm/src/client"
	"github.com/pachyderm/pachyderm/src/client/pfs"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/index"
	"github.com/pachyderm/pachyderm/src/server/pkg/storage/fileset/tar"
	"golang.org/x/net/context"
)

// FileReader is a PFS wrapper for a fileset.MergeReader.
// The primary purpose of this abstraction is to convert from index.Index to
// pfs.FileInfo and to convert a set of index hashes to a file hash.
type FileReader struct {
	file      *pfs.File
	idx       *index.Index
	fmr       *fileset.FileMergeReader
	mr        *fileset.MergeReader
	fileCount int
	hash      hash.Hash
}

func newFileReader(file *pfs.File, idx *index.Index, fmr *fileset.FileMergeReader, mr *fileset.MergeReader) *FileReader {
	h := pfs.NewHash()
	for _, dataRef := range idx.DataOp.DataRefs {
		// TODO Pull from chunk hash.
		h.Write([]byte(dataRef.Hash))
	}
	return &FileReader{
		file: file,
		idx:  idx,
		fmr:  fmr,
		mr:   mr,
		hash: h,
	}
}

func (fr *FileReader) updateFileInfo(idx *index.Index) {
	fr.fileCount++
	for _, dataRef := range idx.DataOp.DataRefs {
		fr.hash.Write([]byte(dataRef.Hash))
	}
}

// Info returns the info for the file.
func (fr *FileReader) Info() *pfs.FileInfoV2 {
	return &pfs.FileInfoV2{
		File: fr.file,
		Hash: pfs.EncodeHash(fr.hash.Sum(nil)),
	}
}

// Get writes a tar stream that contains the file.
func (fr *FileReader) Get(w io.Writer, noPadding ...bool) error {
	if err := fr.fmr.Get(w); err != nil {
		return err
	}
	for fr.fileCount > 0 {
		fmr, err := fr.mr.Next()
		if err != nil {
			return err
		}
		if err := fmr.Get(w); err != nil {
			return err
		}
		fr.fileCount--
	}
	if len(noPadding) > 0 && noPadding[0] {
		return nil
	}
	// Close a tar writer to create tar EOF padding.
	return tar.NewWriter(w).Close()
}

func (fr *FileReader) drain() error {
	for fr.fileCount > 0 {
		if _, err := fr.mr.Next(); err != nil {
			return err
		}
		fr.fileCount--
	}
	return nil
}

func newFileInfoV2FromFile(commit *pfs.Commit, fr fileset.FileReaderAPI) *pfs.FileInfoV2 {
	idx := fr.Index()
	h := pfs.NewHash()
	if idx.DataOp != nil {
		for _, dataRef := range idx.DataOp.DataRefs {
			h.Write([]byte(dataRef.Hash))
		}
	}

	return &pfs.FileInfoV2{
		File: client.NewFile(commit.Repo.Name, commit.ID, idx.Path),
		Hash: pfs.EncodeHash(h.Sum(nil)),
	}
}

// Reader iterates over FileInfoV2s generated from a fileset.ReaderAPI
type Reader struct {
	commit *pfs.Commit
	r1, r2 fileset.ReaderAPI
}

// Creates a Reader which emits FileInfoV2s with the information from commit, and the entries from readers
// returned by getReader.  If getReader returns different Readers all bets are off.
func NewReader(commit *pfs.Commit, getReader func() (fileset.ReaderAPI, error)) (*Reader, error) {
	r1, err := getReader()
	if err != nil {
		return nil, err
	}
	r2, err := getReader()
	if err != nil {
		return nil, err
	}
	return &Reader{
		commit: commit,
		r1:     r1,
		r2:     r2,
	}, nil
}

func (r *Reader) Iterate(ctx context.Context, cb func(*pfs.FileInfoV2, fileset.FileReaderAPI) error) error {
	s1, _ := newStream(r.r1), newStream(r.r2)
	for {
		fr, err := s1.Next()
		if err != nil {
			return err
		}
		if fr == nil {
			return nil
		}

		idx := fr.Index()
		h := pfs.NewHash()
		// TODO: handle directories
		if idx.DataOp != nil {
			for _, dataRef := range idx.DataOp.DataRefs {
				h.Write([]byte(dataRef.Hash))
			}
		}
		finfo := &pfs.FileInfoV2{
			File: client.NewFile(r.commit.Repo.Name, r.commit.ID, idx.Path),
			Hash: pfs.EncodeHash(h.Sum(nil)),
		}
		if err := cb(finfo, fr); err != nil {
			return err
		}
	}
}

type stream struct {
	r fileset.ReaderAPI

	readers chan fileset.FileReaderAPI
	errs    chan error

	next   fileset.FileReaderAPI
	err    error
	isDone bool
}

func newStream(r fileset.ReaderAPI) *stream {
	s := &stream{
		r:       r,
		readers: make(chan fileset.FileReaderAPI),
		errs:    make(chan error),
	}
	go func() {
		if err := s.r.Iterate(func(fr fileset.FileReaderAPI) error {
			s.errs <- nil
			s.readers <- fr
			return nil
		}); err != nil {
			s.errs <- err
		}
		close(s.readers)
		close(s.errs)
	}()
	return s
}

func (s *stream) pullOne() {
	err := <-s.errs
	next, stillOpen := <-s.readers
	if stillOpen {
		s.next, s.err = next, err
	}
}

func (s *stream) Peek() *index.Index {
	if s.next == nil {
		s.pullOne()
	}
	if s.next == nil {
		return nil
	}
	return s.next.Index()
}

func (s *stream) Next() (fileset.FileReaderAPI, error) {
	if s.next == nil {
		s.pullOne()
	}
	ret, err := s.next, s.err
	s.next, s.err = nil, nil
	return ret, err
}
