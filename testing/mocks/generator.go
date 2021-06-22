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

package mocks

type Generator struct {
	GetBalanceFunc      func(symbol string) ([]byte, error)
	TokensDepositedFunc func(symbol string) (string, error)
	TokensWithdrawnFunc func(symbol string) (string, error)
}

func (g *Generator) GetBalance(symbol string) ([]byte, error) {
	return g.GetBalanceFunc(symbol)
}

func (g *Generator) TokensDeposited(symbol string) (string, error) {
	return g.TokensDepositedFunc(symbol)
}

func (g *Generator) TokensWithdrawn(symbol string) (string, error) {
	return g.TokensWithdrawnFunc(symbol)
}
