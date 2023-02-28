package p2p

import (
	karmem "karmem.org/golang"
	"unsafe"
)

var _ unsafe.Pointer

var _Null = make([]byte, 32)
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
	StatusCodeGetPooledTransactionMsg       StatusCode = 15
	StatusCodePooledTransactionsMsg         StatusCode = 16
)

type (
	PacketIdentifier uint64
)

const (
	PacketIdentifierMessage = 14302180353067076632
)

type Message struct {
	Code      StatusCode
	Size      uint32
	Payload   []byte
	ReceiveAt uint32
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
	writer.Write4At(offset, uint32(25))
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
	writer.Write4At(__ReceiveAtOffset, *(*uint32)(unsafe.Pointer(&x.ReceiveAt)))

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
func (x *MessageViewer) ReceiveAt() (v uint32) {
	if 21+4 > x.size() {
		return v
	}
	return *(*uint32)(unsafe.Add(unsafe.Pointer(&x._data), 21))
}
