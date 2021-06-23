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

package errors

import (
	"github.com/optakt/flow-dps/rosetta/meta"
)

// Error represents an error as defined by the Rosetta API specification. It
// contains an error definition, which has an error code, error message and
// retriable flag that never change, as well as a description and a list of
// details to provide more granular error information.
// See: https://www.rosetta-api.org/docs/api_objects.html#error
type Error struct {
	meta.ErrorDefinition
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

func Internal(message string, details ...Detail) Error {
	return newError(ErrorInternal, message, details...)
}

func InvalidFormat(message string, details ...Detail) Error {
	return newError(ErrorInvalidFormat, message, details...)
}

func newError(def meta.ErrorDefinition, message string, details ...Detail) Error {

	err := Error{
		ErrorDefinition: def,
		Description:     message,
	}

	if len(details) > 0 {
		err.Details = make(map[string]interface{})
		for _, d := range details {
			d(&err)
		}
	}

	return err
}
