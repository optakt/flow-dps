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

	// used as reference for index
	prefixFirst = 1
	prefixLast  = 2

	// used for indexing core data of the DPS
	prefixCommit  = 10
	prefixHeader  = 11
	prefixEvents  = 12
	prefixPayload = 13

	// used for indexing auxiliary data of the DPS
	prefixTransaction = 20
	prefixCollection  = 21

	// used for indexing indexes
	prefixHeightForBlock            = 30
	prefixTransactionsForHeight     = 31
	prefixTransactionsForCollection = 32
	prefixCollectionsForHeight      = 33
)
