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

package convert

import (
	"time"

	"github.com/onflow/flow-go/model/flow"
)

func TypesToStrings(types []flow.EventType) []string {
	ss := make([]string, 0, len(types))
	for _, typ := range types {
		ss = append(ss, string(typ))
	}
	return ss
}

func StringsToTypes(ss []string) []flow.EventType {
	types := make([]flow.EventType, 0, len(ss))
	for _, s := range ss {
		types = append(types, flow.EventType(s))
	}
	return types
}

func RosettaTime(t time.Time) int64 {
	return t.UnixNano() / 1_000_000
}
