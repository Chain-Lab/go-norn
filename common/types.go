package common

const (
	HashLength = 32
)

type Hash [HashLength]byte

//func HashBytesToString(h *Hash) string {
//	return hex.EncodeToString(h[:])
//}
