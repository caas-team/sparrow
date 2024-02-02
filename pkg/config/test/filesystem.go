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

type MockFS struct {
	OpenFunc func(name string) (fs.File, error)
}

func (m *MockFS) Open(name string) (fs.File, error) {
	return m.OpenFunc(name)
}

type MockFile struct {
	Content []byte
	readPos int

	CloseFunc func() error
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
	if mf.CloseFunc != nil {
		return mf.CloseFunc()
	}
	return nil
}

func (mf *MockFile) Stat() (fs.FileInfo, error) {
	return nil, nil
}
