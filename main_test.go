package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const appName = "fastest-request"

func TestParseCLIFlagsWithInvalidCEP(t *testing.T) {
	cli, usage, err := ParseCLIFlags(appName, []string{""})
	assert.Nil(t, cli)
	assert.NotEmpty(t, usage)
	assert.ErrorIs(t, err, ErrInvalidCEP)
}

func TestParseCLIFlagsWithInvalidTimeout(t *testing.T) {
	cli, usage, err := ParseCLIFlags(appName, []string{"-cep=cep", "-timeout=error"})
	assert.Nil(t, cli)
	assert.NotEmpty(t, usage)
	assert.ErrorIs(t, err, ErrInvalidTimeout)
}

func TestParseCLIFlagsWithInvalidArgs(t *testing.T) {
	cli, usage, err := ParseCLIFlags(appName, []string{"-x=y"})
	assert.Nil(t, cli)
	assert.NotEmpty(t, usage)
	assert.ErrorIs(t, err, ErrInvalidFlags)
}

func TestGetCEPToTimeout(t *testing.T) {
	resp, err := GetCEP("69999999", 1*time.Microsecond)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGetCEP(t *testing.T) {
	resp, err := GetCEP("69999999", 1*time.Second)
	assert.NotNil(t, resp)
	assert.NoError(t, err)
}
