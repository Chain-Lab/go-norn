package interfaces

import "github.com/syndtr/goleveldb/leveldb"

type DBInterface interface {
	Get([]byte) byte
	Insert([]byte, byte) bool
	Remove([]byte) bool
	BatchInsert(batch *leveldb.Batch)
	BatchRemove()
}
