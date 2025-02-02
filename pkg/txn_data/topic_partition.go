package txn_data

import "slices"

func (t *TopicPartition) Equal(u *TopicPartition) bool {
	return t.Topic == u.Topic && slices.Equal(t.ParNum, u.ParNum)
}

func EqualTopicPartition(a, b *TopicPartition) bool {
	return a.Equal(b)
}
