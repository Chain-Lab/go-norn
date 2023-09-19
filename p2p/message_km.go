package p2p

import (
	karmem "karmem.org/golang"
	"unsafe"
)

var _ unsafe.Pointer

var _Null = make([]byte, 64)
var _NullReader = karmem.NewReader(_Null)

type (
	StatusCode uint8
)

const (
	StatusCodePingMsg                       StatusCode = 0
	StatusCodePongMsg                       StatusCode = 1
	StatusCodeStatusMsg                     StatusCode = 2
	StatusCodeNewBlockHashesMsg             StatusCode = 3
	StatusCodeTransactionsMsg               StatusCode = 4
	StatusCodeGetBlockHeadersMsg            StatusCode = 5
	StatusCodeBlockHeadersMsg               StatusCode = 6
	StatusCodeGetBlockBodiesMsg             StatusCode = 7
	StatusCodeBlockBodiesMsg                StatusCode = 8
	StatusCodeNewBlockMsg                   StatusCode = 9
	StatusCodeGetNodeDataMsg                StatusCode = 10
	StatusCodeNodeDataMsg                   StatusCode = 11
	StatusCodeGetReceiptsMsg                StatusCode = 12
	StatusCodeReceiptsMsg                   StatusCode = 13
	StatusCodeNewPooledTransactionHashesMsg StatusCode = 14
	StatusCodePooledTransactionsMsg         StatusCode = 16
	StatusCodeSyncStatusReq                 StatusCode = 17
	StatusCodeSyncStatusMsg                 StatusCode = 18
	StatusCodeSyncGetBlocksMsg              StatusCode = 19
	StatusCodeSyncBlocksMsg                 StatusCode = 20
	StatusCodeGetBufferedBlocksMsg          StatusCode = 21
	StatusCodeBufferedBlocksMsg             StatusCode = 22
	StatusCodeTimeSyncReq                   StatusCode = 23
	StatusCodeTimeSyncRsp                   StatusCode = 24
)

type (
	PacketIdentifier uint64
)

const (
	PacketIdentifierSyncStatusMsg    = 12064657818327214469
	PacketIdentifierTimeSyncMsg      = 6014709404869090737
	PacketIdentifierMessage          = 14302180353067076632
	PacketIdentifierBroadcastMessage = 3104464370606199534
)

type SyncStatusMsg struct {
	LatestHeight        int64
	LatestHash          [32]byte
	BufferedStartHeight int64
	BufferedEndHeight   int64
}

func NewSyncStatusMsg() SyncStatusMsg {
	return SyncStatusMsg{}
}

func (x *SyncStatusMsg) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierSyncStatusMsg
}

func (x *SyncStatusMsg) Reset() {
	x.Read((*SyncStatusMsgViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *SyncStatusMsg) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *SyncStatusMsg) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(64)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(60))
	__LatestHeightOffset := offset + 4
	writer.Write8At(__LatestHeightOffset, *(*uint64)(unsafe.Pointer(&x.LatestHeight)))
	__LatestHashOffset := offset + 12
	writer.WriteAt(__LatestHashOffset, (*[32]byte)(unsafe.Pointer(&x.LatestHash))[:])
	__BufferedStartHeightOffset := offset + 44
	writer.Write8At(__BufferedStartHeightOffset, *(*uint64)(unsafe.Pointer(&x.BufferedStartHeight)))
	__BufferedEndHeightOffset := offset + 52
	writer.Write8At(__BufferedEndHeightOffset, *(*uint64)(unsafe.Pointer(&x.BufferedEndHeight)))

	return offset, nil
}

func (x *SyncStatusMsg) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewSyncStatusMsgViewer(reader, 0), reader)
}

func (x *SyncStatusMsg) Read(viewer *SyncStatusMsgViewer, reader *karmem.Reader) {
	x.LatestHeight = viewer.LatestHeight()
	__LatestHashSlice := viewer.LatestHash()
	__LatestHashLen := len(__LatestHashSlice)
	copy(x.LatestHash[:], __LatestHashSlice)
	for i := __LatestHashLen; i < len(x.LatestHash); i++ {
		x.LatestHash[i] = 0
	}
	x.BufferedStartHeight = viewer.BufferedStartHeight()
	x.BufferedEndHeight = viewer.BufferedEndHeight()
}

type TimeSyncMsg struct {
	Code       int8
	ReqTime    int64
	RecReqTime int64
	RspTime    int64
	RecRspTime int64
}

func NewTimeSyncMsg() TimeSyncMsg {
	return TimeSyncMsg{}
}

func (x *TimeSyncMsg) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierTimeSyncMsg
}

func (x *TimeSyncMsg) Reset() {
	x.Read((*TimeSyncMsgViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *TimeSyncMsg) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *TimeSyncMsg) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(40)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(37))
	__CodeOffset := offset + 4
	writer.Write1At(__CodeOffset, *(*uint8)(unsafe.Pointer(&x.Code)))
	__ReqTimeOffset := offset + 5
	writer.Write8At(__ReqTimeOffset, *(*uint64)(unsafe.Pointer(&x.ReqTime)))
	__RecReqTimeOffset := offset + 13
	writer.Write8At(__RecReqTimeOffset, *(*uint64)(unsafe.Pointer(&x.RecReqTime)))
	__RspTimeOffset := offset + 21
	writer.Write8At(__RspTimeOffset, *(*uint64)(unsafe.Pointer(&x.RspTime)))
	__RecRspTimeOffset := offset + 29
	writer.Write8At(__RecRspTimeOffset, *(*uint64)(unsafe.Pointer(&x.RecRspTime)))

	return offset, nil
}

func (x *TimeSyncMsg) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewTimeSyncMsgViewer(reader, 0), reader)
}

func (x *TimeSyncMsg) Read(viewer *TimeSyncMsgViewer, reader *karmem.Reader) {
	x.Code = viewer.Code()
	x.ReqTime = viewer.ReqTime()
	x.RecReqTime = viewer.RecReqTime()
	x.RspTime = viewer.RspTime()
	x.RecRspTime = viewer.RecRspTime()
}

type Message struct {
	Code      StatusCode
	Size      uint32
	Payload   []byte
	ReceiveAt int64
}

func NewMessage() Message {
	return Message{}
}

func (x *Message) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierMessage
}

func (x *Message) Reset() {
	x.Read((*MessageViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *Message) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *Message) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(32)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(29))
	__CodeOffset := offset + 4
	writer.Write1At(__CodeOffset, *(*uint8)(unsafe.Pointer(&x.Code)))
	__SizeOffset := offset + 5
	writer.Write4At(__SizeOffset, *(*uint32)(unsafe.Pointer(&x.Size)))
	__PayloadSize := uint(1 * len(x.Payload))
	__PayloadOffset, err := writer.Alloc(__PayloadSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+9, uint32(__PayloadOffset))
	writer.Write4At(offset+9+4, uint32(__PayloadSize))
	writer.Write4At(offset+9+4+4, 1)
	__PayloadSlice := *(*[3]uint)(unsafe.Pointer(&x.Payload))
	__PayloadSlice[1] = __PayloadSize
	__PayloadSlice[2] = __PayloadSize
	writer.WriteAt(__PayloadOffset, *(*[]byte)(unsafe.Pointer(&__PayloadSlice)))
	__ReceiveAtOffset := offset + 21
	writer.Write8At(__ReceiveAtOffset, *(*uint64)(unsafe.Pointer(&x.ReceiveAt)))

	return offset, nil
}

func (x *Message) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewMessageViewer(reader, 0), reader)
}

func (x *Message) Read(viewer *MessageViewer, reader *karmem.Reader) {
	x.Code = StatusCode(viewer.Code())
	x.Size = viewer.Size()
	__PayloadSlice := viewer.Payload(reader)
	__PayloadLen := len(__PayloadSlice)
	if __PayloadLen > cap(x.Payload) {
		x.Payload = append(x.Payload, make([]byte, __PayloadLen-len(x.Payload))...)
	}
	x.Payload = x.Payload[:__PayloadLen]
	copy(x.Payload, __PayloadSlice)
	for i := __PayloadLen; i < len(x.Payload); i++ {
		x.Payload[i] = 0
	}
	x.ReceiveAt = viewer.ReceiveAt()
}

type BroadcastMessage struct {
	ID   []byte
	Data []byte
}

func NewBroadcastMessage() BroadcastMessage {
	return BroadcastMessage{}
}

func (x *BroadcastMessage) PacketIdentifier() PacketIdentifier {
	return PacketIdentifierBroadcastMessage
}

func (x *BroadcastMessage) Reset() {
	x.Read((*BroadcastMessageViewer)(unsafe.Pointer(&_Null)), _NullReader)
}

func (x *BroadcastMessage) WriteAsRoot(writer *karmem.Writer) (offset uint, err error) {
	return x.Write(writer, 0)
}

func (x *BroadcastMessage) Write(writer *karmem.Writer, start uint) (offset uint, err error) {
	offset = start
	size := uint(32)
	if offset == 0 {
		offset, err = writer.Alloc(size)
		if err != nil {
			return 0, err
		}
	}
	writer.Write4At(offset, uint32(28))
	__IDSize := uint(1 * len(x.ID))
	__IDOffset, err := writer.Alloc(__IDSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+4, uint32(__IDOffset))
	writer.Write4At(offset+4+4, uint32(__IDSize))
	writer.Write4At(offset+4+4+4, 1)
	__IDSlice := *(*[3]uint)(unsafe.Pointer(&x.ID))
	__IDSlice[1] = __IDSize
	__IDSlice[2] = __IDSize
	writer.WriteAt(__IDOffset, *(*[]byte)(unsafe.Pointer(&__IDSlice)))
	__DataSize := uint(1 * len(x.Data))
	__DataOffset, err := writer.Alloc(__DataSize)
	if err != nil {
		return 0, err
	}
	writer.Write4At(offset+16, uint32(__DataOffset))
	writer.Write4At(offset+16+4, uint32(__DataSize))
	writer.Write4At(offset+16+4+4, 1)
	__DataSlice := *(*[3]uint)(unsafe.Pointer(&x.Data))
	__DataSlice[1] = __DataSize
	__DataSlice[2] = __DataSize
	writer.WriteAt(__DataOffset, *(*[]byte)(unsafe.Pointer(&__DataSlice)))

	return offset, nil
}

func (x *BroadcastMessage) ReadAsRoot(reader *karmem.Reader) {
	x.Read(NewBroadcastMessageViewer(reader, 0), reader)
}

func (x *BroadcastMessage) Read(viewer *BroadcastMessageViewer, reader *karmem.Reader) {
	__IDSlice := viewer.ID(reader)
	__IDLen := len(__IDSlice)
	if __IDLen > cap(x.ID) {
		x.ID = append(x.ID, make([]byte, __IDLen-len(x.ID))...)
	}
	x.ID = x.ID[:__IDLen]
	copy(x.ID, __IDSlice)
	for i := __IDLen; i < len(x.ID); i++ {
		x.ID[i] = 0
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
}

type SyncStatusMsgViewer struct {
	_data [64]byte
}

func NewSyncStatusMsgViewer(reader *karmem.Reader, offset uint32) (v *SyncStatusMsgViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*SyncStatusMsgViewer)(unsafe.Pointer(&_Null))
	}
	v = (*SyncStatusMsgViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*SyncStatusMsgViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *SyncStatusMsgViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *SyncStatusMsgViewer) LatestHeight() (v int64) {
	if 4+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 4))
}
func (x *SyncStatusMsgViewer) LatestHash() (v []byte) {
	if 12+32 > x.size() {
		return []byte{}
	}
	slice := [3]uintptr{
		uintptr(unsafe.Add(unsafe.Pointer(&x._data), 12)), 32, 32,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *SyncStatusMsgViewer) BufferedStartHeight() (v int64) {
	if 44+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 44))
}
func (x *SyncStatusMsgViewer) BufferedEndHeight() (v int64) {
	if 52+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 52))
}

type TimeSyncMsgViewer struct {
	_data [40]byte
}

func NewTimeSyncMsgViewer(reader *karmem.Reader, offset uint32) (v *TimeSyncMsgViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*TimeSyncMsgViewer)(unsafe.Pointer(&_Null))
	}
	v = (*TimeSyncMsgViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*TimeSyncMsgViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *TimeSyncMsgViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *TimeSyncMsgViewer) Code() (v int8) {
	if 4+1 > x.size() {
		return v
	}
	return *(*int8)(unsafe.Add(unsafe.Pointer(&x._data), 4))
}
func (x *TimeSyncMsgViewer) ReqTime() (v int64) {
	if 5+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 5))
}
func (x *TimeSyncMsgViewer) RecReqTime() (v int64) {
	if 13+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 13))
}
func (x *TimeSyncMsgViewer) RspTime() (v int64) {
	if 21+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 21))
}
func (x *TimeSyncMsgViewer) RecRspTime() (v int64) {
	if 29+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 29))
}

type MessageViewer struct {
	_data [32]byte
}

func NewMessageViewer(reader *karmem.Reader, offset uint32) (v *MessageViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*MessageViewer)(unsafe.Pointer(&_Null))
	}
	v = (*MessageViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*MessageViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *MessageViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *MessageViewer) Code() (v StatusCode) {
	if 4+1 > x.size() {
		return v
	}
	return *(*StatusCode)(unsafe.Add(unsafe.Pointer(&x._data), 4))
}
func (x *MessageViewer) Size() (v uint32) {
	if 5+4 > x.size() {
		return v
	}
	return *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 5))
}
func (x *MessageViewer) Payload(reader *karmem.Reader) (v []byte) {
	if 9+12 > x.size() {
		return []byte{}
	}
	offset := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 9))
	size := *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 9+4))
	if !reader.IsValidOffset(offset, size) {
		return []byte{}
	}
	length := uintptr(size / 1)
	slice := [3]uintptr{
		uintptr(unsafe.Add(reader.Pointer, offset)), length, length,
	}
	return *(*[]byte)(unsafe.Pointer(&slice))
}
func (x *MessageViewer) ReceiveAt() (v int64) {
	if 21+8 > x.size() {
		return v
	}
	return *(*int64)(unsafe.Add(unsafe.Pointer(&x._data), 21))
}

type BroadcastMessageViewer struct {
	_data [32]byte
}

func NewBroadcastMessageViewer(reader *karmem.Reader, offset uint32) (v *BroadcastMessageViewer) {
	if !reader.IsValidOffset(offset, 8) {
		return (*BroadcastMessageViewer)(unsafe.Pointer(&_Null))
	}
	v = (*BroadcastMessageViewer)(unsafe.Add(reader.Pointer, offset))
	if !reader.IsValidOffset(offset, v.size()) {
		return (*BroadcastMessageViewer)(unsafe.Pointer(&_Null))
	}
	return v
}

func (x *BroadcastMessageViewer) size() uint32 {
	return *(*uint32)(unsafe.Pointer(&x._data))
}
func (x *BroadcastMessageViewer) ID(reader *karmem.Reader) (v []byte) {
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
func (x *BroadcastMessageViewer) Data(reader *karmem.Reader) (v []byte) {
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
