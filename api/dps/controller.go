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

package dps

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/models/dps"
)

type Controller struct {
	state dps.State
}

// NewController creates a controller using the given state store.
func NewController(state dps.State) *Controller {
	c := &Controller{
		state: state,
	}
	return c
}

func (c *Controller) GetHeader(optional *uint64) (*flow.Header, uint64, error) {
	height := c.state.Last().Height()
	if optional != nil {
		height = *optional
	}
	header, err := c.state.Data().Header(height)
	if err != nil {
		return nil, 0, fmt.Errorf("could not return header: %w", err)
	}
	return header, height, nil
}

func (c *Controller) ReadRegisters(optional *uint64, paths []ledger.Path) ([]ledger.Value, uint64, error) {
	height := c.state.Last().Height()
	if optional != nil {
		height = *optional
	}
	values := make([]ledger.Value, 0, len(paths))
	for _, path := range paths {
		value, err := c.state.Raw().WithHeight(height).Get(path[:])
		if err != nil {
			return nil, 0, fmt.Errorf("could not retrieve value (path: %x): %w", path, err)
		}
		values = append(values, ledger.Value(value))
	}
	return values, height, nil
}
