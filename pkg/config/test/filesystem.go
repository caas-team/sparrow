// sparrow
// (C) 2024, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package test

import (
	"io"
	"io/fs"
)

// MockFS provides a mock implementation of the fs.FS interface.
type MockFS struct {
	// OpenFunc allows for customizing the behavior of the Open method.
	OpenFunc func(name string) (fs.File, error)
}

// Open calls the OpenFunc field of the MockFS struct.
func (m *MockFS) Open(name string) (fs.File, error) {
	return m.OpenFunc(name)
}

// MockFile is a mock implementation of the fs.File interface.
type MockFile struct {
	// Content simulates the content of the file. Read operations will return data from this slice.
	Content []byte
	// readPos tracks the current position in Content, simulating the file's read pointer.
	readPos int

	// CloseFunc is an optional function that simulates closing the file. It allows users to
	// specify custom behavior for the Close method, including simulating errors.
	CloseFunc func() error
}

// Read attempts to read bytes from the MockFile into b. It simulates reading by copying bytes
// from mf.Content into b, starting from the current read position.
// Returns the number of bytes read and an error, if any. Once all content has been read, subsequent calls will return io.EOF.
func (mf *MockFile) Read(b []byte) (int, error) {
	if mf.readPos >= len(mf.Content) {
		return 0, io.EOF
	}
	n := copy(b, mf.Content[mf.readPos:])
	mf.readPos += n
	return n, nil
}

// Close simulates closing the file.
func (mf *MockFile) Close() error {
	if mf.CloseFunc != nil {
		return mf.CloseFunc()
	}
	return nil
}

func (mf *MockFile) Stat() (fs.FileInfo, error) {
	return nil, nil
}
