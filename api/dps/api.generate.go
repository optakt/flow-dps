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

//go:generate protoc -I . -I /usr/local/include -I $HOME/.local/include -I $GOPATH/pkg/mod/github.com/srikrsna/protoc-gen-gotag@v0.6.1 --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative  --go-grpc_opt=require_unimplemented_servers=false ./api.proto
//go:generate protoc -I . -I /usr/local/include -I $HOME/.local/include -I $GOPATH/pkg/mod/github.com/srikrsna/protoc-gen-gotag@v0.6.1 --gotag_out=:. --gotag_opt=paths=source_relative ./api.proto

package dps
