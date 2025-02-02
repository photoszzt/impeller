package txn_data

import "slices"

func (a *TxnMetadata) Equal(b *TxnMetadata) bool {
	return a.State == b.State && slices.EqualFunc(a.TopicPartitions, b.TopicPartitions, EqualTopicPartition)
}

func (a *TxnMetaMsg) Equal(b *TxnMetaMsg) bool {
	return a.TransactionalId == b.TransactionalId && a.State == b.State &&
		a.ProdId.TaskEpoch == b.ProdId.TaskEpoch && a.ProdId.TaskId == b.ProdId.TaskId &&
		slices.EqualFunc(a.TopicPartitions, b.TopicPartitions, EqualTopicPartition)
}
