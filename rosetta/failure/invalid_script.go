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

package failure

import (
	"fmt"
)

// InvalidScript is the error for an invalid transaction script.
type InvalidScript struct {
	Description Description
	Script      string
}

// Error implements the error interface.
func (i InvalidScript) Error() string {
	// We don't want to print the entire script, that would be gigantic.
	return fmt.Sprintf("invalid transaction script: %s", i.Description)
}
