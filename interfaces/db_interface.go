package interfaces

import "github.com/syndtr/goleveldb/leveldb"

type DBInterface interface {
	GetGet(key []byte) ([]byte, error)
	InsertInsert(key []byte, value []byte) error
	Remove([]byte) bool
	BatchInsert(batch *leveldb.Batch)
	BatchRemove()
}
