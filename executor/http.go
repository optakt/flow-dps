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

package executor

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

type httpError struct {
	Message string `json:"message"`
	Err     string `json:"error,omitempty"`
}

func (e httpError) Error() string {
	if e.Err == "" {
		return e.Message
	}
	return fmt.Sprintf("%v (err: %v)", e.Message, e.Err)
}

func newHTTPError(code int, message string, err error) *echo.HTTPError {
	e := httpError{
		Message: message,
		Err:     err.Error(),
	}

	return echo.NewHTTPError(code, e)
}
