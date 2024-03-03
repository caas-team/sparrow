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

package interactor

import (
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote/git"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote/gitlab"
)

// Config contains the configuration for the available remote interactors
type Config struct {
	// Git contains the configuration for the git interactor
	Git git.Config `yaml:"git" mapstructure:"git"`
	// Gitlab contains the configuration for the gitlab interactor
	Gitlab gitlab.Config `yaml:"gitlab" mapstructure:"gitlab"`
}

type Type string

func (t Type) Interactor(cfg *Config) remote.Interactor {
	switch t {
	case "git":
		return git.New(cfg.Git)
	case "gitlab":
		return gitlab.New(cfg.Gitlab)
	}
	return nil
}
