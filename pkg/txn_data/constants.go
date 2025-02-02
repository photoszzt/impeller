package txn_data

import (
	"math"
)

const (
	LogTagReserveBits = 3

	PartitionBits = 8
	PartitionMask = uint64(math.MaxUint64) - (1<<PartitionBits - 1)
	TagAndParMask = uint64(math.MaxUint64) - (1<<(PartitionBits+LogTagReserveBits) - 1)
)

const (
	Fence = iota
	MarkLowBits
	ScaleFenceLowBits
	CtrlMetaLowBits
	AbortLowBits
	PreCommitLowBits
	ChkptLowBits
)

const (
	SCALE_FENCE_KEY = "__scale_fence"
)

func FenceTag(nameHash uint64, parNum uint8) uint64 {
	return nameHash&TagAndParMask + uint64(parNum)<<LogTagReserveBits + Fence
}

func ScaleFenceTag(nameHash uint64, parNum uint8) uint64 {
	return nameHash&TagAndParMask + uint64(parNum)<<LogTagReserveBits + ScaleFenceLowBits
}

func MarkerTag(nameHash uint64, par uint8) uint64 {
	return nameHash&TagAndParMask + uint64(par)<<LogTagReserveBits + MarkLowBits
}

func CtrlMetaTag(nameHash uint64, par uint8) uint64 {
	return nameHash&TagAndParMask + uint64(par)<<LogTagReserveBits + CtrlMetaLowBits
}

func TxnAbortTag(nameHash uint64, par uint8) uint64 {
	return nameHash&TagAndParMask + uint64(par)<<LogTagReserveBits + AbortLowBits
}

func TxnPreCommitTag(nameHash uint64, par uint8) uint64 {
	return nameHash&TagAndParMask + uint64(par)<<LogTagReserveBits + PreCommitLowBits
}

func ChkptTag(nameHash uint64, par uint8) uint64 {
	return nameHash&TagAndParMask + uint64(par)<<LogTagReserveBits + ChkptLowBits
}
