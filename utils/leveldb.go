package utils

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"sync"
)

var (
	databaseOnce sync.Once
	levelDBInst  *LevelDB
	db           *leveldb.DB
)

func GetLevelDBInst() *LevelDB {
	databaseOnce.Do(func() {
		levelDBInst = &LevelDB{}
		var err error
		db, err = leveldb.OpenFile("./data", nil)

		if err != nil {
			log.WithField("error", err).Errorln("Open leveldb failed.")
			return
		}
	})
	return levelDBInst
}

type LevelDB struct{}

func (ld *LevelDB) Get(key []byte) ([]byte, error) {
	return db.Get(key, nil)
}

func (ld *LevelDB) Insert(key []byte, value []byte) error {
	return db.Put(key, value, nil)
}

func (ld *LevelDB) Remove(key []byte) error {
	return db.Delete(key, nil)
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

	return db.Write(batch, nil)
}

func (ld *LevelDB) BatchDelete(key [][]byte) error {
	batch := new(leveldb.Batch)
	for idx := range key {
		batch.Delete(key[idx])
	}

	return db.Write(batch, nil)
}
