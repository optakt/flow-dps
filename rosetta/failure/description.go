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
	"strconv"
	"strings"

	"github.com/onflow/flow-go/model/flow"
)

type Description struct {
	Text   string
	Fields Fields
}

func NewDescription(text string, fields ...FieldFunc) Description {
	d := Description{
		Text:   text,
		Fields: []Field{},
	}
	for _, field := range fields {
		field(&d.Fields)
	}
	return d
}

func (d Description) String() string {
	if len(d.Fields) == 0 {
		return d.Text
	}
	return fmt.Sprintf("%s (%s)", d.Text, d.Fields)
}

type Field struct {
	Key string
	Val interface{}
}

type Fields []Field

func (f Fields) Iterate(handle func(key string, val interface{})) {
	for _, field := range f {
		handle(field.Key, field.Val)
	}
}

func (f Fields) String() string {
	parts := make([]string, 0, len(f))
	for _, field := range f {
		part := fmt.Sprintf("%s: %s", field.Key, field.Val)
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

type FieldFunc func(*Fields)

func WithErr(err error) FieldFunc {
	return func(f *Fields) {
		field := Field{Key: "error", Val: err.Error()}
		*f = append(*f, field)
	}
}

func WithInt(key string, val int) FieldFunc {
	return func(f *Fields) {
		field := Field{Key: key, Val: strconv.FormatInt(int64(val), 10)}
		*f = append(*f, field)
	}
}

func WithUint64(key string, val uint64) FieldFunc {
	return func(f *Fields) {
		field := Field{Key: key, Val: strconv.FormatUint(val, 10)}
		*f = append(*f, field)
	}
}

func WithID(key string, val flow.Identifier) FieldFunc {
	return func(f *Fields) {
		field := Field{Key: key, Val: val}
		*f = append(*f, field)
	}
}

func WithString(key string, val string) FieldFunc {
	return func(f *Fields) {
		field := Field{Key: key, Val: val}
		*f = append(*f, field)
	}
}

func WithStrings(key string, vals ...string) FieldFunc {
	return func(f *Fields) {
		field := Field{Key: key, Val: vals}
		*f = append(*f, field)
	}
}
