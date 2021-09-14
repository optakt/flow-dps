package transactor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNew(t *testing.T) {
	validate := mocks.BaselineValidator(t)
	generate := mocks.BaselineGenerator(t)
	invoke := mocks.BaselineInvoker(t)
	submit := mocks.BaselineSubmitter(t)

	tr := New(validate, generate, invoke, submit)

	assert.Equal(t, validate, tr.validate)
	assert.Equal(t, generate, tr.generate)
	assert.Equal(t, invoke, tr.invoke)
	assert.Equal(t, submit, tr.submit)
}

func BaselineTransactor(t *testing.T, opts ...func(*Transactor)) *Transactor {
	t.Helper()

	tr := Transactor{
		validate: mocks.BaselineValidator(t),
		generate: mocks.BaselineGenerator(t),
		invoke:   mocks.BaselineInvoker(t),
		submit:   mocks.BaselineSubmitter(t),
	}

	for _, opt := range opts {
		opt(&tr)
	}

	return &tr
}

func WithValidator(validator Validator) func(*Transactor) {
	return func(transactor *Transactor) {
		transactor.validate = validator
	}
}

func WithGenerator(generator Generator) func(*Transactor) {
	return func(transactor *Transactor) {
		transactor.generate = generator
	}
}

func WithInvoker(invoker Invoker) func(*Transactor) {
	return func(transactor *Transactor) {
		transactor.invoke = invoker
	}
}

func WithSubmitter(submitter Submitter) func(*Transactor) {
	return func(transactor *Transactor) {
		transactor.submit = submitter
	}
}
