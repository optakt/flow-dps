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

package validator

import (
	"github.com/go-playground/validator/v10"

	"github.com/optakt/flow-dps/models/dps"
)

type Validator struct {
	params   dps.Params
	index    dps.Reader
	validate *validator.Validate
}

func New(params dps.Params, index dps.Reader, config Configuration) *Validator {

	v := &Validator{
		params:   params,
		index:    index,
		validate: newRequestValidator(config),
	}

	return v
}
