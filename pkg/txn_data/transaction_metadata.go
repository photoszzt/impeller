//go:generate msgp
//msgp:ignore TxnMetadataJSONSerde TxnMetadataMsgpSerde
package txn_data

import (
	"fmt"
)

type TransactionState uint32

const (
	EMPTY = TransactionState(iota) // transaction has not existed yet
	BEGIN                          // transaction has started and ongoing
	PREPARE_COMMIT
	PREPARE_ABORT
	COMPLETE_COMMIT
	COMPLETE_ABORT
	FENCE
)

func (ts TransactionState) String() string {
	return []string{"EMPTY", "BEGIN", "PREPARE_COMMIT", "PREPARE_ABORT", "COMPLETE_COMMIT", "COMPLETE_ABORT", "FENCE"}[ts]
}

func (ts TransactionState) IsValidPreviousState(prevState TransactionState) bool {
	switch ts {
	case EMPTY:
		return prevState == EMPTY || prevState == COMPLETE_ABORT || prevState == COMPLETE_COMMIT || prevState == FENCE
	case BEGIN:
		return prevState == BEGIN || prevState == EMPTY || prevState == COMPLETE_ABORT || prevState == COMPLETE_COMMIT || prevState == FENCE
	case PREPARE_COMMIT:
		return prevState == BEGIN
	case PREPARE_ABORT:
		return prevState == BEGIN || prevState == FENCE
	case COMPLETE_ABORT:
		return prevState == PREPARE_ABORT
	case COMPLETE_COMMIT:
		return prevState == PREPARE_COMMIT
	case FENCE:
		return prevState == EMPTY
	default:
		panic(fmt.Sprintf("transaction state is not recognized: %d", ts))
	}
}

// log entries in transaction log
type TxnMetadata struct {
	TopicPartitions []*TopicPartition `json:"tp,omitempty" msg:"tp,omitempty"`
	State           TransactionState  `json:"st,omitempty" msg:"st,omitempty"`
}
