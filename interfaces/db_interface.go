package interfaces

type DBInterface interface {
	Get(key []byte) ([]byte, error)
	Insert(key []byte, value []byte) error
	Remove(key []byte) error
	BatchInsert(key [][]byte, value [][]byte) error
	BatchDelete(key [][]byte) error
}
