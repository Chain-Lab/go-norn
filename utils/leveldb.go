package utils

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

type LevelDB struct {
	db *leveldb.DB
}

func NewLevelDB(path string) (*LevelDB, error) {
	db, err := leveldb.OpenFile(path, nil)

	if err != nil {
		log.WithField("error", err).Errorln("Open leveldb failed.")
		return nil, err
	}

	return &LevelDB{db: db}, nil
}

func (ld *LevelDB) Get(key []byte) ([]byte, error) {
	return ld.db.Get(key, nil)
}

func (ld *LevelDB) Insert(key []byte, value []byte) error {
	return ld.db.Put(key, value, nil)
}

func (ld *LevelDB) Remove(key []byte) error {
	return ld.db.Delete(key, nil)
}

func (ld *LevelDB) BatchInsert(key [][]byte, value [][]byte) error {
	if len(key) != len(value) {
		log.Errorln("Key/Value length not match.")
		return errors.New("Batch insert failed.")
	}

	batch := new(leveldb.Batch)
	for idx := range key {
		batch.Put(key[idx], value[idx])
	}

	return ld.db.Write(batch, nil)
}

func (ld *LevelDB) BatchDelete(key [][]byte) error {
	batch := new(leveldb.Batch)
	for idx := range key {
		batch.Delete(key[idx])
	}

	return ld.db.Write(batch, nil)
}
