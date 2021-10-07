package main

const dictionaryTemplate = `
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

package zbor

// {{ .Name }}Dictionary is a byte slice that contains the result of running the Zstandard training mode
// on the {{ .Name | lower }} of the DPS index. This allows zstandard to achieve a better compression ratio, specifically for
// small data.
// See http://facebook.github.io/zstd/#small-data
// See https://github.com/facebook/zstd/blob/master/doc/zstd_compression_format.md#dictionary-format
var {{ .Name }}Dictionary = []byte{
	{{ range .Bytes }}{{ . }}, {{ end }}
}
`