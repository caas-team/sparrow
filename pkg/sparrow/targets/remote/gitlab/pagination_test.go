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

package gitlab

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNextLink(t *testing.T) {
	type header struct {
		noLinkHeader bool
		key          string
		value        string
	}
	tests := []struct {
		name   string
		header header
		want   string
	}{
		{
			"no link header present",
			header{
				noLinkHeader: true,
			},
			"",
		},
		{
			"no next link in link header present",
			header{
				key:   "link",
				value: "<https://link.first.de>; rel=\"first\", <https://link.last.de>; rel=\"last\"",
			},
			"",
		},
		{
			"link header syntax not valid",
			header{
				key:   "link",
				value: "no link here",
			},
			"",
		},
		{
			"valid next link",
			header{
				key:   "link",
				value: "<https://link.next.de>; rel=\"next\", <https://link.first.de>; rel=\"first\", <https://link.last.de>; rel=\"last\"",
			},
			"https://link.next.de",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHeader := http.Header{}
			testHeader.Add(tt.header.key, tt.header.value)

			assert.Equal(t, tt.want, getNextLink(testHeader))
		})
	}
}
