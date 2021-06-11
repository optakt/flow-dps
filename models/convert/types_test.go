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

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/model/flow"
)

func TestTypesToStrings(t *testing.T) {
	t1 := "test1"
	t2 := "test2"
	t3 := "test3"
	t4 := "test4"

	typ1 := flow.EventType(t1)
	typ2 := flow.EventType(t2)
	typ3 := flow.EventType(t3)
	typ4 := flow.EventType(t4)

	ss := TypesToStrings([]flow.EventType{typ1, typ2, typ3, typ4})

	assert.Equal(t, []string{t1, t2, t3,t4}, ss)
}
