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
	"strings"
)

type Description struct {
	Text   string
	Fields FieldList
}

func NewDescription(text string, fields ...FieldFunc) Description {
	d := Description{
		Text:   text,
		Fields: FieldList{},
	}
	for _, field := range fields {
		field(&d.Fields)
	}
	return d
}

func (d Description) String() string {
	if len(d.Fields.fields) == 0 {
		return d.Text
	}
	return fmt.Sprintf("%s (%s)", d.Text, d.Fields)
}

type Field struct {
	Key string
	Val interface{}
}

type FieldList struct {
	fields []Field
}

func (f FieldList) Iterate(handle func(key string, val interface{})) {
	for _, field := range f.fields {
		handle(field.Key, field.Val)
	}
}

func (f FieldList) String() string {
	parts := make([]string, 0, len(f.fields))
	for _, field := range f.fields {
		part := fmt.Sprintf("%s: %s", field.Key, field.Val)
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

type FieldFunc func(*FieldList)

func WithErr(err error) FieldFunc {
	return func(f *FieldList) {
		field := Field{Key: "error", Val: err.Error()}
		f.fields = append(f.fields, field)
	}
}

func WithInt(key string, val int) FieldFunc {
	return func(f *FieldList) {
		field := Field{Key: key, Val: val}
		f.fields = append(f.fields, field)
	}
}

func WithUint64(key string, val uint64) FieldFunc {
	return func(f *FieldList) {
		field := Field{Key: key, Val: val}
		f.fields = append(f.fields, field)
	}
}

func WithString(key string, val string) FieldFunc {
	return func(f *FieldList) {
		field := Field{Key: key, Val: val}
		f.fields = append(f.fields, field)
	}
}

func WithStrings(key string, vals ...string) FieldFunc {
	return func(f *FieldList) {
		field := Field{Key: key, Val: vals}
		f.fields = append(f.fields, field)
	}
}
