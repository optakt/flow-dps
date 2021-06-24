// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package rosetta

import (
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/meta"
)

// Configuration represents the configuration parameters of a particular blockchain from
// the Rosetta API's perspective. It details some blockchain metadata, its supported operations,
// errors, and more.
type Configuration interface {
	Network() identifier.Network
	Version() meta.Version
	Operations() []string
	Statuses() []meta.StatusDefinition
	Errors() []meta.ErrorDefinition
	Check(network identifier.Network) error
}
