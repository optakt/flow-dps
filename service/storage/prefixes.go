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

package storage

const (
	PrefixFirst = 1
	PrefixLast  = 2

	PrefixHeightForBlock       = 7
	PrefixHeightForTransaction = 16

	PrefixCommit  = 4
	PrefixHeader  = 3
	PrefixEvents  = 5
	PrefixPayload = 6

	PrefixTransaction = 8
	PrefixCollection  = 10
	PrefixGuarantee   = 17

	PrefixTransactionsForHeight     = 9
	PrefixTransactionsForCollection = 12
	PrefixCollectionsForHeight      = 11
	PrefixResults                   = 13

	PrefixSeal           = 14
	PrefixSealsForHeight = 15
)
