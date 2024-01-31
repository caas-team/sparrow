package test

import (
	"io"
	"io/fs"
)

type MockFS struct {
	OpenFunc func(name string) (fs.File, error)
}

func (m *MockFS) Open(name string) (fs.File, error) {
	return m.OpenFunc(name)
}

type MockFile struct {
	Content []byte
	readPos int
}

func (mf *MockFile) Read(b []byte) (int, error) {
	if mf.readPos >= len(mf.Content) {
		return 0, io.EOF
	}
	n := copy(b, mf.Content[mf.readPos:])
	mf.readPos += n
	return n, nil
}

func (mf *MockFile) Close() error {
	return nil
}

func (mf *MockFile) Stat() (fs.FileInfo, error) {
	return nil, nil
}
