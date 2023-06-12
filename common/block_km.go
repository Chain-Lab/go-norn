package common

import (
	karmem "karmem.org/golang"
	"unsafe"
)

var _ unsafe.Pointer

var _Null = make([]byte, 208)
var _NullReader = karmem.NewReader(_Null)

type (
	PacketIdentifier uint64
)

const (
	PacketIdentifierTransactionBody = 12338327753349604039
	PacketIdentifierTransaction     = 11306821277700167240
	PacketIdentifierGenesisParams   = 6807263723591883565
	PacketIdentifierGeneralParams   = 13469299839833990814
	PacketIdentifierBlockHeader     = 6888014730382219470
	PacketIdentifierBlock           = 1202114546008698459
)

type TransactionBody struct {
	Hash      [32]byte
	Signature []byte
	Address   [20]byte
	Public    [33]byte
	Data      []byte
	Expire    int64
	Timestamp int64
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
	size := uint(136)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(129))
	__HashOffset := offset + 4
	writer.WriteAt(__HashOffset, (*[32]byte)(unsafe.Pointer(&x.Hash))[:])
	__SignatureSize := uint(1 * len(x.Signature))
	__SignatureOffset, err := writer.Alloc(__SignatureSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+36, uint32(__SignatureOffset))
	writer.Write4At(offset+36+4, uint32(__SignatureSize))
	writer.Write4At(offset+36+4+4, 1)
	__SignatureSlice := *(*[3]uint)(unsafe.Pointer(&x.Signature))
	__SignatureSlice[1] = __SignatureSize
	__SignatureSlice[2] = __SignatureSize
	writer.WriteAt(__SignatureOffset, *(*[]byte)(unsafe.Pointer(&__SignatureSlice)))
	__AddressOffset := offset + 48
	writer.WriteAt(__AddressOffset, (*[20]byte)(unsafe.Pointer(&x.Address))[:])
	__PublicOffset := offset + 68
	writer.WriteAt(__PublicOffset, (*[33]byte)(unsafe.Pointer(&x.Public))[:])
	__DataSize := uint(1 * len(x.Data))
	__DataOffset, err := writer.Alloc(__DataSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+101, uint32(__DataOffset))
	writer.Write4At(offset+101+4, uint32(__DataSize))
	writer.Write4At(offset+101+4+4, 1)
	__DataSlice := *(*[3]uint)(unsafe.Pointer(&x.Data))
	__DataSlice[1] = __DataSize
	__DataSlice[2] = __DataSize
	writer.WriteAt(__DataOffset, *(*[]byte)(unsafe.Pointer(&__DataSlice)))
	__ExpireOffset := offset + 113
	writer.Write8At(__ExpireOffset, *(*uint64)(unsafe.Pointer(&x.Expire)))
	__TimestampOffset := offset + 121
	writer.Write8At(__TimestampOffset, *(*uint64)(unsafe.Pointer(&x.Timestamp)))

	return offset, nil
}

func (x *TransactionBody) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewTransactionBodyViewer(reader, 0), reader)
}

func (x *TransactionBody) Read(viewer *TransactionBodyViewer, reader *karmem.Reader) {
	__HashSlice := viewer.Hash()
	__HashLen := len(__HashSlice)
	copy(x.Hash[:], __HashSlice)
	for i := __HashLen; i < len(x.Hash); i++ {
		x.Hash[i] = 0
	}
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
	__BodySize := uint(136)
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

type GenesisParams struct {
	Order       [128]byte
	TimeParam   int64
	Seed        [32]byte
	VerifyParam [32]byte
}

func NewGenesisParams() GenesisParams {
	return GenesisParams{}
}

func (x *GenesisParams) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierGenesisParams
}

func (x *GenesisParams) Reset() {
	x.Read((*GenesisParamsViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *GenesisParams) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *GenesisParams) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(208)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(204))
	__OrderOffset := offset + 4
	writer.WriteAt(__OrderOffset, (*[128]byte)(unsafe.Pointer(&x.Order))[:])
	__TimeParamOffset := offset + 132
	writer.Write8At(__TimeParamOffset, *(*uint64)(unsafe.Pointer(&x.TimeParam)))
	__SeedOffset := offset + 140
	writer.WriteAt(__SeedOffset, (*[32]byte)(unsafe.Pointer(&x.Seed))[:])
	__VerifyParamOffset := offset + 172
	writer.WriteAt(__VerifyParamOffset, (*[32]byte)(unsafe.Pointer(&x.VerifyParam))[:])

	return offset, nil
}

func (x *GenesisParams) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewGenesisParamsViewer(reader, 0), reader)
}

func (x *GenesisParams) Read(viewer *GenesisParamsViewer, reader *karmem.Reader) {
	__OrderSlice := viewer.Order()
	__OrderLen := len(__OrderSlice)
	copy(x.Order[:], __OrderSlice)
	for i := __OrderLen; i < len(x.Order); i++ {
		x.Order[i] = 0
	}
	x.TimeParam = viewer.TimeParam()
	__SeedSlice := viewer.Seed()
	__SeedLen := len(__SeedSlice)
	copy(x.Seed[:], __SeedSlice)
	for i := __SeedLen; i < len(x.Seed); i++ {
		x.Seed[i] = 0
	}
	__VerifyParamSlice := viewer.VerifyParam()
	__VerifyParamLen := len(__VerifyParamSlice)
	copy(x.VerifyParam[:], __VerifyParamSlice)
	for i := __VerifyParamLen; i < len(x.VerifyParam); i++ {
		x.VerifyParam[i] = 0
	}
}

type GeneralParams struct {
	Result       []byte
	Proof        []byte
	RandomNumber [33]byte
	S            []byte
	T            []byte
}

func NewGeneralParams() GeneralParams {
	return GeneralParams{}
}

func (x *GeneralParams) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierGeneralParams
}

func (x *GeneralParams) Reset() {
	x.Read((*GeneralParamsViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *GeneralParams) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *GeneralParams) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(88)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(85))
	__ResultSize := uint(1 * len(x.Result))
	__ResultOffset, err := writer.Alloc(__ResultSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+4, uint32(__ResultOffset))
	writer.Write4At(offset+4+4, uint32(__ResultSize))
	writer.Write4At(offset+4+4+4, 1)
	__ResultSlice := *(*[3]uint)(unsafe.Pointer(&x.Result))
	__ResultSlice[1] = __ResultSize
	__ResultSlice[2] = __ResultSize
	writer.WriteAt(__ResultOffset, *(*[]byte)(unsafe.Pointer(&__ResultSlice)))
	__ProofSize := uint(1 * len(x.Proof))
	__ProofOffset, err := writer.Alloc(__ProofSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+16, uint32(__ProofOffset))
	writer.Write4At(offset+16+4, uint32(__ProofSize))
	writer.Write4At(offset+16+4+4, 1)
	__ProofSlice := *(*[3]uint)(unsafe.Pointer(&x.Proof))
	__ProofSlice[1] = __ProofSize
	__ProofSlice[2] = __ProofSize
	writer.WriteAt(__ProofOffset, *(*[]byte)(unsafe.Pointer(&__ProofSlice)))
	__RandomNumberOffset := offset + 28
	writer.WriteAt(__RandomNumberOffset, (*[33]byte)(unsafe.Pointer(&x.RandomNumber))[:])
	__SSize := uint(1 * len(x.S))
	__SOffset, err := writer.Alloc(__SSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+61, uint32(__SOffset))
	writer.Write4At(offset+61+4, uint32(__SSize))
	writer.Write4At(offset+61+4+4, 1)
	__SSlice := *(*[3]uint)(unsafe.Pointer(&x.S))
	__SSlice[1] = __SSize
	__SSlice[2] = __SSize
	writer.WriteAt(__SOffset, *(*[]byte)(unsafe.Pointer(&__SSlice)))
	__TSize := uint(1 * len(x.T))
	__TOffset, err := writer.Alloc(__TSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+73, uint32(__TOffset))
	writer.Write4At(offset+73+4, uint32(__TSize))
	writer.Write4At(offset+73+4+4, 1)
	__TSlice := *(*[3]uint)(unsafe.Pointer(&x.T))
	__TSlice[1] = __TSize
	__TSlice[2] = __TSize
	writer.WriteAt(__TOffset, *(*[]byte)(unsafe.Pointer(&__TSlice)))

	return offset, nil
}

func (x *GeneralParams) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewGeneralParamsViewer(reader, 0), reader)
}

func (x *GeneralParams) Read(viewer *GeneralParamsViewer, reader *karmem.Reader) {
	__ResultSlice := viewer.Result(reader)
	__ResultLen := len(__ResultSlice)
	if __ResultLen > cap(x.Result) {
		x.Result = append(x.Result, make([]byte, __ResultLen-len(x.Result))...)
	}
	x.Result = x.Result[:__ResultLen]
	copy(x.Result, __ResultSlice)
	for i := __ResultLen; i < len(x.Result); i++ {
		x.Result[i] = 0
	}
	__ProofSlice := viewer.Proof(reader)
	__ProofLen := len(__ProofSlice)
	if __ProofLen > cap(x.Proof) {
		x.Proof = append(x.Proof, make([]byte, __ProofLen-len(x.Proof))...)
	}
	x.Proof = x.Proof[:__ProofLen]
	copy(x.Proof, __ProofSlice)
	for i := __ProofLen; i < len(x.Proof); i++ {
		x.Proof[i] = 0
	}
	__RandomNumberSlice := viewer.RandomNumber()
	__RandomNumberLen := len(__RandomNumberSlice)
	copy(x.RandomNumber[:], __RandomNumberSlice)
	for i := __RandomNumberLen; i < len(x.RandomNumber); i++ {
		x.RandomNumber[i] = 0
	}
	__SSlice := viewer.S(reader)
	__SLen := len(__SSlice)
	if __SLen > cap(x.S) {
		x.S = append(x.S, make([]byte, __SLen-len(x.S))...)
	}
	x.S = x.S[:__SLen]
	copy(x.S, __SSlice)
	for i := __SLen; i < len(x.S); i++ {
		x.S[i] = 0
	}
	__TSlice := viewer.T(reader)
	__TLen := len(__TSlice)
	if __TLen > cap(x.T) {
		x.T = append(x.T, make([]byte, __TLen-len(x.T))...)
	}
	x.T = x.T[:__TLen]
	copy(x.T, __TSlice)
	for i := __TLen; i < len(x.T); i++ {
		x.T[i] = 0
	}
}

type BlockHeader struct {
	Timestamp     int64
	PrevBlockHash [32]byte
	BlockHash     [32]byte
	MerkleRoot    [32]byte
	Height        int64
	PublicKey     [33]byte
	Params        []byte
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
	size := uint(168)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(161))
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
	__PublicKeyOffset := offset + 116
	writer.WriteAt(__PublicKeyOffset, (*[33]byte)(unsafe.Pointer(&x.PublicKey))[:])
	__ParamsSize := uint(1 * len(x.Params))
	__ParamsOffset, err := writer.Alloc(__ParamsSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+149, uint32(__ParamsOffset))
	writer.Write4At(offset+149+4, uint32(__ParamsSize))
	writer.Write4At(offset+149+4+4, 1)
	__ParamsSlice := *(*[3]uint)(unsafe.Pointer(&x.Params))
	__ParamsSlice[1] = __ParamsSize
	__ParamsSlice[2] = __ParamsSize
	writer.WriteAt(__ParamsOffset, *(*[]byte)(unsafe.Pointer(&__ParamsSlice)))

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
	__PublicKeySlice := viewer.PublicKey()
	__PublicKeyLen := len(__PublicKeySlice)
	copy(x.PublicKey[:], __PublicKeySlice)
	for i := __PublicKeyLen; i < len(x.PublicKey); i++ {
		x.PublicKey[i] = 0
	}
	__ParamsSlice := viewer.Params(reader)
	__ParamsLen := len(__ParamsSlice)
	if __ParamsLen > cap(x.Params) {
		x.Params = append(x.Params, make([]byte, __ParamsLen-len(x.Params))...)
	}
	x.Params = x.Params[:__ParamsLen]
	copy(x.Params, __ParamsSlice)
	for i := __ParamsLen; i < len(x.Params); i++ {
		x.Params[i] = 0
	}
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
	__HeaderSize := uint(168)
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
	_data [136]byte
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
func (x *TransactionBodyViewer) Hash() (v []byte) {
	if 4+32 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 4)), 32, 32,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *TransactionBodyViewer) Signature(reader *karmem.Reader) (v []byte) {
	if 36+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 36))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 36+4))
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
	if 48+20 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 48)), 20, 20,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *TransactionBodyViewer) Public() (v []byte) {
	if 68+33 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 68)), 33, 33,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *TransactionBodyViewer) Data(reader *karmem.Reader) (v []byte) {
	if 101+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 101))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 101+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *TransactionBodyViewer) Expire() (v int64) {
	if 113+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 113))
}
func (x *TransactionBodyViewer) Timestamp() (v int64) {
	if 121+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 121))
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

type GenesisParamsViewer struct {
	_data [208]byte
}

func NewGenesisParamsViewer(reader *karmem.Reader, offset uint32) (v *GenesisParamsViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*GenesisParamsViewer)(unsafe.Pointer(&_Null))
	}
	v = (*GenesisParamsViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*GenesisParamsViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *GenesisParamsViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *GenesisParamsViewer) Order() (v []byte) {
	if 4+128 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 4)), 128, 128,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *GenesisParamsViewer) TimeParam() (v int64) {
	if 132+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 132))
}
func (x *GenesisParamsViewer) Seed() (v []byte) {
	if 140+32 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 140)), 32, 32,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *GenesisParamsViewer) VerifyParam() (v []byte) {
	if 172+32 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 172)), 32, 32,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}

type GeneralParamsViewer struct {
	_data [88]byte
}

func NewGeneralParamsViewer(reader *karmem.Reader, offset uint32) (v *GeneralParamsViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*GeneralParamsViewer)(unsafe.Pointer(&_Null))
	}
	v = (*GeneralParamsViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*GeneralParamsViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *GeneralParamsViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *GeneralParamsViewer) Result(reader *karmem.Reader) (v []byte) {
	if 4+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 4))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 4+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *GeneralParamsViewer) Proof(reader *karmem.Reader) (v []byte) {
	if 16+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 16))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 16+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *GeneralParamsViewer) RandomNumber() (v []byte) {
	if 28+33 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 28)), 33, 33,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *GeneralParamsViewer) S(reader *karmem.Reader) (v []byte) {
	if 61+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 61))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 61+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *GeneralParamsViewer) T(reader *karmem.Reader) (v []byte) {
	if 73+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 73))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 73+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}

type BlockHeaderViewer struct {
	_data [168]byte
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
func (x *BlockHeaderViewer) Timestamp() (v int64) {
	if 4+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 4))
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
func (x *BlockHeaderViewer) Height() (v int64) {
	if 108+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 108))
}
func (x *BlockHeaderViewer) PublicKey() (v []byte) {
	if 116+33 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 116)), 33, 33,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *BlockHeaderViewer) Params(reader *karmem.Reader) (v []byte) {
	if 149+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 149))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 149+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
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
