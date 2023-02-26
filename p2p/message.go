package p2p

import (
	"bufio"
	karmem "karmem.org/golang"
	"sync"

	log "github.com/sirupsen/logrus"
)

// 这个 writerPool 暂时这样用，后续再看如何修改
var writerPool = sync.Pool{New: func() any { return karmem.NewWriter(1024) }}

// Send 方法用于提供一个通用的消息发送接口
func Send(w bufio.Writer, msgcode StatusCode, writer karmem.Writer) error {
	msgWriter := writerPool.Get().(*karmem.Writer)
	payload := writer.Bytes()
	// todo: 这里数据的大小暂时留空，作为冗余字段
	// todo: 观察一下这里处理数据会不会有较高的耗时，特别是数据比较大的情况下
	msg := Message{
		Code:      msgcode,
		Size:      uint32(len(payload)),
		Payload:   payload,
		ReceiveAt: 0,
	}

	if _, err := msg.WriteAsRoot(msgWriter); err != nil {
		log.WithField("error", err).Debugln("Encode data failed.")
		return err
	}

	msgBytes := msgWriter.Bytes()
	_, err := w.Write(msgBytes)
	if err != nil {
		log.WithField("error", err).Debugln("Send data to peer errored.")
		return err
	}

	msgWriter.Reset()
	writerPool.Put(msgWriter)

	return nil
}
