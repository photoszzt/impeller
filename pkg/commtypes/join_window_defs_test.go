package commtypes

import (
	"testing"
	"time"
)

const (
	TEST_JWS_SIZE       = 123
	TEST_JWS_OTHER_SIZE = 456
)

func TestBeforeShouldNotModifyGrace(t *testing.T) {
	jws, err := NewJoinWindowsWithGrace(time.Duration(TEST_JWS_SIZE)*time.Millisecond,
		time.Duration(TEST_JWS_OTHER_SIZE)*time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	jws, err = jws.Before(time.Duration(TEST_JWS_SIZE) * time.Second)
	if err != nil {
		t.Fatal(err.Error())
	}
	if jws.GracePeriodMs() != TEST_JWS_OTHER_SIZE {
		t.Fatal("should equal")
	}
}

func TestAfterShouldNotModifyGrace(t *testing.T) {
	jws, err := NewJoinWindowsWithGrace(time.Duration(TEST_JWS_SIZE)*time.Millisecond,
		time.Duration(TEST_JWS_OTHER_SIZE)*time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	jws, err = jws.After(time.Duration(TEST_JWS_SIZE) * time.Second)
	if err != nil {
		t.Fatal(err.Error())
	}
	if jws.GracePeriodMs() != TEST_JWS_OTHER_SIZE {
		t.Fatal("should equal")
	}
}
