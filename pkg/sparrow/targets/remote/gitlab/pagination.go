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
	"strings"
)

const (
	linkHeader = "Link"
	linkNext   = "next"
)

// getNextLink returns the url to the next page of
// a paginated http response provided in the passed response header.
func getNextLink(header http.Header) string {
	link := header.Get(linkHeader)
	if link == "" {
		return ""
	}

	for _, link := range strings.Split(link, ",") {
		linkParts := strings.Split(link, ";")
		if len(linkParts) != 2 {
			continue
		}
		linkType := strings.Trim(strings.Split(linkParts[1], "=")[1], "\"")

		if linkType != linkNext {
			continue
		}
		return strings.Trim(linkParts[0], "< >")
	}
	return ""
}
