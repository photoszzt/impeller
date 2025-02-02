package tests

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/rs/zerolog/log"
)

type MockTesting struct {
	testing.TB
	testName string
	failed   bool
}

var _ = testing.TB(&MockTesting{})

func (t *MockTesting) Cleanup(func()) {
	panic("not implemented")
}
func (t *MockTesting) Error(args ...interface{}) {
	log.Error().Msg(fmt.Sprint(args...))
}

func (t *MockTesting) Errorf(format string, args ...interface{}) {
	log.Error().Msgf(format, args...)
}

func (t *MockTesting) Fail() {
	t.failed = true
	log.Fatal().Msg("Failed")
}

func (t *MockTesting) FailNow() {
	t.failed = true
	log.Fatal().Msg(fmt.Sprintf("%s Failed\n", t.testName))
	runtime.Goexit()
}

func (t *MockTesting) Failed() bool {
	return t.failed
}

func (t *MockTesting) Fatal(args ...interface{}) {
	log.Fatal().Msg(fmt.Sprintln(args...))
	t.FailNow()
}

func (t *MockTesting) Fatalf(format string, args ...interface{}) {
	log.Fatal().Msg(fmt.Sprintf(format, args...))
	t.FailNow()
}
func (t *MockTesting) Helper() {
	log.Info().Msg(t.testName)
}
func (t *MockTesting) Log(args ...interface{}) {
	log.Info().Msg(fmt.Sprintln(args...))
}
func (t *MockTesting) Logf(format string, args ...interface{}) {
	log.Info().Msg(fmt.Sprintf(format, args...))
}
func (t *MockTesting) Name() string {
	return t.testName
}

func (t *MockTesting) Setenv(key, value string) {
	panic("not implemented")
}
func (t *MockTesting) Skip(args ...interface{}) {
	panic("not implemented")
}
func (t *MockTesting) SkipNow() {
	panic("not implemented")
}
func (t *MockTesting) Skipf(format string, args ...interface{}) {
	panic("not implemented")
}
func (t *MockTesting) Skipped() bool {
	panic("not implemented")
}
func (t *MockTesting) TempDir() string {
	panic("not implemented")
}
