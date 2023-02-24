package core

import (
	karmem "karmem.org/golang"
	"unsafe"
)

var _ unsafe.Pointer

var _Null = make([]byte, 120)
var _NullReader = karmem.NewReader(_Null)

type (
	PacketIdentifier uint64
)

const (
	PacketIdentifierTransactionBody = 12338327753349604039
	PacketIdentifierTransaction     = 11306821277700167240
	PacketIdentifierBlockHeader     = 6888014730382219470
	PacketIdentifierBlock           = 1202114546008698459
)

type TransactionBody struct {
	Signature []byte
	Address   [20]byte
	Public    [33]byte
	Data      []byte
	Expire    uint64
	Timestamp uint64
}

func NewTransactionBody() TransactionBody {
	return TransactionBody{}
}

func (x *TransactionBody) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierTransactionBody
}

func (x *TransactionBody) Reset() {
	x.Read((*TransactionBodyViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *TransactionBody) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *TransactionBody) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(104)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(97))
	__SignatureSize := uint(1 * len(x.Signature))
	__SignatureOffset, err := writer.Alloc(__SignatureSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+4, uint32(__SignatureOffset))
	writer.Write4At(offset+4+4, uint32(__SignatureSize))
	writer.Write4At(offset+4+4+4, 1)
	__SignatureSlice := *(*[3]uint)(unsafe.Pointer(&x.Signature))
	__SignatureSlice[1] = __SignatureSize
	__SignatureSlice[2] = __SignatureSize
	writer.WriteAt(__SignatureOffset, *(*[]byte)(unsafe.Pointer(&__SignatureSlice)))
	__AddressOffset := offset + 16
	writer.WriteAt(__AddressOffset, (*[20]byte)(unsafe.Pointer(&x.Address))[:])
	__PublicOffset := offset + 36
	writer.WriteAt(__PublicOffset, (*[33]byte)(unsafe.Pointer(&x.Public))[:])
	__DataSize := uint(1 * len(x.Data))
	__DataOffset, err := writer.Alloc(__DataSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+69, uint32(__DataOffset))
	writer.Write4At(offset+69+4, uint32(__DataSize))
	writer.Write4At(offset+69+4+4, 1)
	__DataSlice := *(*[3]uint)(unsafe.Pointer(&x.Data))
	__DataSlice[1] = __DataSize
	__DataSlice[2] = __DataSize
	writer.WriteAt(__DataOffset, *(*[]byte)(unsafe.Pointer(&__DataSlice)))
	__ExpireOffset := offset + 81
	writer.Write8At(__ExpireOffset, *(*uint64)(unsafe.Pointer(&x.Expire)))
	__TimestampOffset := offset + 89
	writer.Write8At(__TimestampOffset, *(*uint64)(unsafe.Pointer(&x.Timestamp)))

	return offset, nil
}

func (x *TransactionBody) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewTransactionBodyViewer(reader, 0), reader)
}

func (x *TransactionBody) Read(viewer *TransactionBodyViewer, reader *karmem.Reader) {
	__SignatureSlice := viewer.Signature(reader)
	__SignatureLen := len(__SignatureSlice)
	if __SignatureLen > cap(x.Signature) {
		x.Signature = append(x.Signature, make([]byte, __SignatureLen-len(x.Signature))...)
	}
	x.Signature = x.Signature[:__SignatureLen]
	copy(x.Signature, __SignatureSlice)
	for i := __SignatureLen; i < len(x.Signature); i++ {
		x.Signature[i] = 0
	}
	__AddressSlice := viewer.Address()
	__AddressLen := len(__AddressSlice)
	copy(x.Address[:], __AddressSlice)
	for i := __AddressLen; i < len(x.Address); i++ {
		x.Address[i] = 0
	}
	__PublicSlice := viewer.Public()
	__PublicLen := len(__PublicSlice)
	copy(x.Public[:], __PublicSlice)
	for i := __PublicLen; i < len(x.Public); i++ {
		x.Public[i] = 0
	}
	__DataSlice := viewer.Data(reader)
	__DataLen := len(__DataSlice)
	if __DataLen > cap(x.Data) {
		x.Data = append(x.Data, make([]byte, __DataLen-len(x.Data))...)
	}
	x.Data = x.Data[:__DataLen]
	copy(x.Data, __DataSlice)
	for i := __DataLen; i < len(x.Data); i++ {
		x.Data[i] = 0
	}
	x.Expire = viewer.Expire()
	x.Timestamp = viewer.Timestamp()
}

type Transaction struct {
	Body TransactionBody
}

func NewTransaction() Transaction {
	return Transaction{}
}

func (x *Transaction) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierTransaction
}

func (x *Transaction) Reset() {
	x.Read((*TransactionViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *Transaction) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *Transaction) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(8)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	__BodySize := uint(104)
	__BodyOffset, err := writer.Alloc(__BodySize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+0, uint32(__BodyOffset))
	if _, err := x.Body.Write(writer, __BodyOffset); err != nil {
		return offset, err
	}

	return offset, nil
}

func (x *Transaction) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewTransactionViewer(reader, 0), reader)
}

func (x *Transaction) Read(viewer *TransactionViewer, reader *karmem.Reader) {
	x.Body.Read(viewer.Body(reader), reader)
}

type BlockHeader struct {
	Timestamp     uint64
	PrevBlockHash [32]byte
	BlockHash     [32]byte
	MerkleRoot    [32]byte
	Height        uint64
}

func NewBlockHeader() BlockHeader {
	return BlockHeader{}
}

func (x *BlockHeader) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierBlockHeader
}

func (x *BlockHeader) Reset() {
	x.Read((*BlockHeaderViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *BlockHeader) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *BlockHeader) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(120)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(116))
	__TimestampOffset := offset + 4
	writer.Write8At(__TimestampOffset, *(*uint64)(unsafe.Pointer(&x.Timestamp)))
	__PrevBlockHashOffset := offset + 12
	writer.WriteAt(__PrevBlockHashOffset, (*[32]byte)(unsafe.Pointer(&x.PrevBlockHash))[:])
	__BlockHashOffset := offset + 44
	writer.WriteAt(__BlockHashOffset, (*[32]byte)(unsafe.Pointer(&x.BlockHash))[:])
	__MerkleRootOffset := offset + 76
	writer.WriteAt(__MerkleRootOffset, (*[32]byte)(unsafe.Pointer(&x.MerkleRoot))[:])
	__HeightOffset := offset + 108
	writer.Write8At(__HeightOffset, *(*uint64)(unsafe.Pointer(&x.Height)))

	return offset, nil
}

func (x *BlockHeader) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewBlockHeaderViewer(reader, 0), reader)
}

func (x *BlockHeader) Read(viewer *BlockHeaderViewer, reader *karmem.Reader) {
	x.Timestamp = viewer.Timestamp()
	__PrevBlockHashSlice := viewer.PrevBlockHash()
	__PrevBlockHashLen := len(__PrevBlockHashSlice)
	copy(x.PrevBlockHash[:], __PrevBlockHashSlice)
	for i := __PrevBlockHashLen; i < len(x.PrevBlockHash); i++ {
		x.PrevBlockHash[i] = 0
	}
	__BlockHashSlice := viewer.BlockHash()
	__BlockHashLen := len(__BlockHashSlice)
	copy(x.BlockHash[:], __BlockHashSlice)
	for i := __BlockHashLen; i < len(x.BlockHash); i++ {
		x.BlockHash[i] = 0
	}
	__MerkleRootSlice := viewer.MerkleRoot()
	__MerkleRootLen := len(__MerkleRootSlice)
	copy(x.MerkleRoot[:], __MerkleRootSlice)
	for i := __MerkleRootLen; i < len(x.MerkleRoot); i++ {
		x.MerkleRoot[i] = 0
	}
	x.Height = viewer.Height()
}

type Block struct {
	Header       BlockHeader
	Transactions []Transaction
}

func NewBlock() Block {
	return Block{}
}

func (x *Block) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierBlock
}

func (x *Block) Reset() {
	x.Read((*BlockViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *Block) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *Block) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(24)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(20))
	__HeaderSize := uint(120)
	__HeaderOffset, err := writer.Alloc(__HeaderSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+4, uint32(__HeaderOffset))
	if _, err := x.Header.Write(writer, __HeaderOffset); err != nil {
		return offset, err
	}
	__TransactionsSize := uint(8 * len(x.Transactions))
	__TransactionsOffset, err := writer.Alloc(__TransactionsSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+8, uint32(__TransactionsOffset))
	writer.Write4At(offset+8+4, uint32(__TransactionsSize))
	writer.Write4At(offset+8+4+4, 8)
	for i := range x.Transactions {
		if _, err := x.Transactions[i].Write(writer, __TransactionsOffset); err != nil {
			return offset, err
		}
		__TransactionsOffset += 8
	}

	return offset, nil
}

func (x *Block) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewBlockViewer(reader, 0), reader)
}

func (x *Block) Read(viewer *BlockViewer, reader *karmem.Reader) {
	x.Header.Read(viewer.Header(reader), reader)
	__TransactionsSlice := viewer.Transactions(reader)
	__TransactionsLen := len(__TransactionsSlice)
	if __TransactionsLen > cap(x.Transactions) {
		x.Transactions = append(x.Transactions, make([]Transaction, __TransactionsLen-len(x.Transactions))...)
	}
	x.Transactions = x.Transactions[:__TransactionsLen]
	for i := range x.Transactions {
		x.Transactions[i].Read(&__TransactionsSlice[i], reader)
	}
}

type TransactionBodyViewer struct {
	_data [104]byte
}

func NewTransactionBodyViewer(reader *karmem.Reader, offset uint32) (v *TransactionBodyViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*TransactionBodyViewer)(unsafe.Pointer(&_Null))
	}
	v = (*TransactionBodyViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*TransactionBodyViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *TransactionBodyViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *TransactionBodyViewer) Signature(reader *karmem.Reader) (v []byte) {
	if 4+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 4))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 4+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	if length > 73 {
		length = 73
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *TransactionBodyViewer) Address() (v []byte) {
	if 16+20 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 16)), 20, 20,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *TransactionBodyViewer) Public() (v []byte) {
	if 36+33 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 36)), 33, 33,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *TransactionBodyViewer) Data(reader *karmem.Reader) (v []byte) {
	if 69+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 69))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 69+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *TransactionBodyViewer) Expire() (v uint64) {
	if 81+8 > x.size() {
		return v
	}
	return *(*uint64)(unsafe.Add(unsafe.Pointer(&x._data), 81))
}
func (x *TransactionBodyViewer) Timestamp() (v uint64) {
	if 89+8 > x.size() {
		return v
	}
	return *(*uint64)(unsafe.Add(unsafe.Pointer(&x._data), 89))
}

type TransactionViewer struct {
	_data [8]byte
}

func NewTransactionViewer(reader *karmem.Reader, offset uint32) (v *TransactionViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*TransactionViewer)(unsafe.Pointer(&_Null))
	}
	v = (*TransactionViewer)(unsafe.Add(reader.Pointer, offset))
	return v
}

func (x *TransactionViewer) size() uint32 {
	return 8
}
func (x *TransactionViewer) Body(reader *karmem.Reader) (v *TransactionBodyViewer) {
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 0))
	return NewTransactionBodyViewer(reader, offset)
}

type BlockHeaderViewer struct {
	_data [120]byte
}

func NewBlockHeaderViewer(reader *karmem.Reader, offset uint32) (v *BlockHeaderViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*BlockHeaderViewer)(unsafe.Pointer(&_Null))
	}
	v = (*BlockHeaderViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*BlockHeaderViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *BlockHeaderViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *BlockHeaderViewer) Timestamp() (v uint64) {
	if 4+8 > x.size() {
		return v
	}
	return *(*uint64)(unsafe.Add(unsafe.Pointer(&x._data), 4))
}
func (x *BlockHeaderViewer) PrevBlockHash() (v []byte) {
	if 12+32 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 12)), 32, 32,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *BlockHeaderViewer) BlockHash() (v []byte) {
	if 44+32 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 44)), 32, 32,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *BlockHeaderViewer) MerkleRoot() (v []byte) {
	if 76+32 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 76)), 32, 32,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *BlockHeaderViewer) Height() (v uint64) {
	if 108+8 > x.size() {
		return v
	}
	return *(*uint64)(unsafe.Add(unsafe.Pointer(&x._data), 108))
}

type BlockViewer struct {
	_data [24]byte
}

func NewBlockViewer(reader *karmem.Reader, offset uint32) (v *BlockViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*BlockViewer)(unsafe.Pointer(&_Null))
	}
	v = (*BlockViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*BlockViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *BlockViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *BlockViewer) Header(reader *karmem.Reader) (v *BlockHeaderViewer) {
	if 4+4 > x.size() {
		return (*BlockHeaderViewer)(unsafe.Pointer(&_Null))
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 4))
	return NewBlockHeaderViewer(reader, offset)
}
func (x *BlockViewer) Transactions(reader *karmem.Reader) (v []TransactionViewer) {
	if 8+12 > x.size() {
		return []TransactionViewer{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 8))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 8+4))
	if !reader.IsValidOffset(offset, size) {
		return []TransactionViewer{}
	}
	length := uintptr(size / 8)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]TransactionViewer)(unsafe.Pointer(&slice))
}
