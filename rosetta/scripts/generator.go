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

package scripts

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/optakt/flow-dps/models/dps"
)

type Generator struct {
	params         dps.Params
	getBalance     *template.Template
	transferTokens *template.Template
}

func NewGenerator(params dps.Params) *Generator {
	g := Generator{
		params:         params,
		getBalance:     template.Must(template.New("get_balance").Parse(getBalance)),
		transferTokens: template.Must(template.New("transfer_tokens").Parse(transferTokens)),
	}
	return &g
}

func (g *Generator) GetBalance(symbol string) ([]byte, error) {
	return g.compile(g.getBalance, symbol)
}

func (g *Generator) TransferTokens(symbol string) ([]byte, error) {
	return g.compile(g.transferTokens, symbol)
}

func (g *Generator) compile(template *template.Template, symbol string) ([]byte, error) {
	token, ok := g.params.Tokens[symbol]
	if !ok {
		return nil, fmt.Errorf("invalid token symbol (%s)", symbol)
	}
	data := struct {
		Params dps.Params
		Token  dps.Token
	}{
		Params: g.params,
		Token:  token,
	}
	buf := &bytes.Buffer{}
	err := template.Execute(buf, data)
	if err != nil {
		return nil, fmt.Errorf("could not execute template: %w", err)
	}
	script := buf.Bytes()
	return script, nil
}
