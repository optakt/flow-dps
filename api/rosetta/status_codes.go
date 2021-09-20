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
	"net/http"
)

// The Rosetta API specification expects every error returned from the Rosetta
// API to be a HTTP status code 500 (internal server error). We optionally make
// it possible to have a more expressive API by returning meaningful HTTP status
// codes where appropriate.
var (
	statusOK                  = http.StatusOK
	statusBadRequest          = http.StatusInternalServerError
	statusUnprocessableEntity = http.StatusInternalServerError
	statusInternalServerError = http.StatusInternalServerError
)

// EnableSmartCodes overwrites the global variables that determine which error
// codes the Rosetta API returns. While we avoid global variables in general,
// these function more as a proxy to the constants of the HTTP package, with the
// ability to change their value.
func EnableSmartCodes() {
	statusOK = http.StatusOK
	statusBadRequest = http.StatusBadRequest
	statusUnprocessableEntity = http.StatusUnprocessableEntity
	statusInternalServerError = http.StatusInternalServerError
}
