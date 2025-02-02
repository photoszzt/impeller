package store

type TABLE_TYPE uint8

const (
	IN_MEM  TABLE_TYPE = 0
	MONGODB TABLE_TYPE = 1
)

type VersionedKeyG[K any] struct {
	Key     K
	Version uint32
}
