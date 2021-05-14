// Copyright 2021 Alvalor S.A.
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

package state

import (
	"fmt"

	"github.com/onflow/flow-go/model/flow"
)

type Chain struct {
	core *Core
}

func (c *Chain) Header(height uint64) (*flow.Header, error) {
	// FIXME: implement header retrieval
	return nil, fmt.Errorf("not implemented")
}

func (c *Chain) Events(height uint64) ([]flow.Event, error) {
	// FIXME: move events retrieval
	return nil, fmt.Errorf("not implemented")
}
