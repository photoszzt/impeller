package common_errors

import (
	"golang.org/x/xerrors"
)

var (
	ErrEmptyPayload                = xerrors.New("payload cannot be empty")
	ErrStreamEmpty                 = xerrors.New("stream empty")
	ErrStreamTimeout               = xerrors.New("blocking pop timeout")
	ErrInvalidStateTransition      = xerrors.New("invalid state transition")
	ErrStreamSourceTimeout         = xerrors.New("SharedLogStreamSource consume timeout")
	ErrShouldExitForScale          = xerrors.New("should exit")
	ErrUnrecognizedSerdeFormat     = xerrors.New("Unrecognized serde format")
	ErrFailToGetOutputSubstream    = xerrors.New("Fail to get output substream")
	ErrReturnDueToTest             = xerrors.New("return due to test")
	ErrReachCutoffPos              = xerrors.New("reach cutoff pos")
	ErrChkptMarkerInvalidGuarantee = xerrors.New("Got checkpoint marker but guarantee is not checkpoint")
	ErrTopicNotFound               = xerrors.New("Topic not found")
	ErrStaleProducer               = xerrors.New("Stale producer")
	ErrInvalidTxnMngrRpc           = xerrors.New("Unrecognized txn rpc type")
)

func IsStreamEmptyError(err error) bool {
	return err == ErrStreamEmpty
}

func IsStreamTimeoutError(err error) bool {
	return err == ErrStreamTimeout
}
