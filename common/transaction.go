package common

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"github.com/chain-lab/go-chronos/metrics"
	log "github.com/sirupsen/logrus"
	karmem "karmem.org/golang"
	"time"
)

//var writerPool = sync.Pool{New: func() any { return karmem.NewWriter(1024) }}

// Verify 交易验证方法
func (tx *Transaction) Verify() bool {
	//writer := writerPool.Get().(*karmem.Writer)
	//defer writerPool.Put(writer)
	//defer writer.Reset()
	// 初始化交易序列化的 writer
	verifyStart := time.Now()
	writer := karmem.NewWriter(1024)

	txBody := tx.Body
	byteSignature := make([]byte, len(txBody.Signature))
	bytePublicKey := txBody.Public[:]
	// todo: 这样将 [32]byte 转换为 []byte 是否存在风险
	byteHash := make([]byte, 32)

	// 对交易的签名进行拷贝，便于后续验证
	copy(byteSignature, txBody.Signature[:])
	copy(byteHash, txBody.Hash[:])

	// 对公钥进行反序列化，得到椭圆曲线上的点 (x, y) 表示公钥
	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), bytePublicKey[:])
	publicKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	if _, err := txBody.WriteAsRoot(writer); err != nil {
		log.WithField("error", err).Debugln("Karmem writer error.")
		return false
	}

	// 为了数据安全起见， 验证的过程先复制一份，然后在复制上面读取
	txByteData := writer.Bytes()
	txBodyCopy := new(TransactionBody)
	txBodyCopy.ReadAsRoot(karmem.NewReader(txByteData))

	// 将哈希和签名设置为空
	txBodyCopy.Hash = [32]byte{}
	txBodyCopy.Signature = []byte{}

	writer.Reset()
	if _, err := txBodyCopy.WriteAsRoot(writer); err != nil {
		log.WithField("error", err).Debugln("Karmem writer error.")
		return false
	}

	hash := sha256.New()
	hash.Write(writer.Bytes())
	txHashBytes := hash.Sum(nil)

	if bytes.Compare(txHashBytes, byteHash) != 0 {
		log.WithFields(log.Fields{
			"hash":  hex.EncodeToString(byteHash)[:8],
			"local": hex.EncodeToString(txHashBytes),
		}).Debugln("Transaction hash not match.")
		return false
	}

	result := ecdsa.VerifyASN1(&publicKey, txHashBytes, byteSignature)
	metrics.VerifyTransactionMetricsSet(float64(time.Since(verifyStart).Milliseconds()))

	return result
}
