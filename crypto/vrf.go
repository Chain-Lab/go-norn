/**
  @author: decision
  @date: 2023/4/17
  @note: VRF 计算及验证函数
**/

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/gookit/config/v2"
	log "github.com/sirupsen/logrus"
	"math/big"
)

const (
	CONSENSUS_FLOOR = 0.0 // 共识要求的最低概率
)

var (
	tt260 = BigPow(2, 260)
)

func VRFCalculate(curve elliptic.Curve, msg []byte) ([]byte, *big.Int, *big.Int, error) {
	N := curve.Params().N
	prvHex := config.String("consensus.prv")
	prv, err := decodePrivateKeyFromHexString(prvHex)
	if err != nil {
		log.WithField("error", err).Fatalln("Load private key failed.")
		return nil, nil, nil, err
	}

	// todo: 消息需要哈希一下，避免由于输入消息过短出现安全问题
	xM, yM := curve.ScalarBaseMult(msg)
	r, err := rand.Int(rand.Reader, N)

	if err != nil {
		log.WithField("err", err).Errorln("Generate random number in VRF failed.")
		return nil, nil, nil, err
	}

	xRm, yRm := curve.ScalarMult(xM, yM, r.Bytes())   // Calculate rM
	xRo, yRo := curve.ScalarBaseMult(r.Bytes())       // Calculate rO
	xV, yV := curve.ScalarMult(xM, yM, prv.D.Bytes()) // V = kO

	sBytes := elliptic.MarshalCompressed(curve, xRm, yRm) // s = marshal(rM)
	oBytes := elliptic.MarshalCompressed(curve, xRo, yRo) // o = marshal(rO)

	// 这里可以改为一个 Hash 函数
	s, o := new(big.Int), new(big.Int)
	s.SetBytes(sBytes)
	o.SetBytes(oBytes)
	s = s.Mul(s, o)
	s = s.Mod(s, N) // s = s * o

	sk := new(big.Int)
	sk = sk.Mul(s, prv.D) // sk = s * k

	t := new(big.Int)
	t = t.Sub(r, sk)
	t = t.Mod(t, N) // t = (r - sk) mod N

	rBytes := elliptic.MarshalCompressed(curve, xV, yV)
	return rBytes, s, t, nil
}

// VRFCheckOutputConsensus 检查一个 VRF 的输出是否满足共识
func VRFCheckOutputConsensus(randomOutput []byte) bool {
	r := new(big.Int)
	r.SetBytes(randomOutput)

	base := new(big.Int)
	base.SetInt64(1000)
	r = r.Mul(r, base)
	r = r.Div(r, tt260)

	prob := float64(r.Int64()) / 1000.0
	return prob > CONSENSUS_FLOOR
}

// VRFCheckLocalConsensus 检查当前节点是否是一个共识节点
func VRFCheckLocalConsensus(vdfOutput []byte) (bool, error) {
	rBytes, _, _, _ := VRFCalculate(elliptic.P256(), vdfOutput)

	return VRFCheckOutputConsensus(rBytes), nil
}

// VRFCheckRemoteConsensus 检查一个其他节点的输出是否满足共识条件
func VRFCheckRemoteConsensus(key *ecdsa.PublicKey, vdfMsg []byte, s *big.Int, t *big.Int, value []byte) (bool, error) {
	verified, err := VRFVerify(elliptic.P256(), key, vdfMsg, s, t, value)
	if !verified || err != nil {
		// 验证过程中出错， 认为验证失败
		return false, nil
	}

	return VRFCheckOutputConsensus(value), nil
}

func VRFVerify(curve elliptic.Curve, key *ecdsa.PublicKey, msg []byte, s *big.Int, t *big.Int, value []byte) (bool, error) {
	N := curve.Params().N
	xR, yR := elliptic.UnmarshalCompressed(curve, value) // random number -> point V

	// todo: 消息需要哈希一下，避免由于输入消息过短出现安全问题
	xM, yM := curve.ScalarBaseMult(msg) // message -> point M

	xTm, yTm := curve.ScalarMult(xM, yM, t.Bytes()) // t * M
	xTo, yTo := curve.ScalarBaseMult(t.Bytes())
	xSv, ySv := curve.ScalarMult(xR, yR, s.Bytes())
	xSy, ySy := curve.ScalarMult(key.X, key.Y, s.Bytes())

	xU1, yU1 := curve.Add(xTm, yTm, xSv, ySv) // u1 = tM + sV = rM
	xU2, yU2 := curve.Add(xTo, yTo, xSy, ySy) // u2 = tO + sY = rO

	sBytes := elliptic.MarshalCompressed(curve, xU1, yU1) // s = marshal(rM)
	oBytes := elliptic.MarshalCompressed(curve, xU2, yU2) // o = marshal(rO)

	// 这里可以改为一个 Hash 函数
	s1, o := new(big.Int), new(big.Int)
	s1.SetBytes(sBytes)
	o.SetBytes(oBytes)
	s1 = s1.Mul(s1, o)
	s1 = s1.Mod(s1, N) // s = s * o

	return s.Cmp(s1) == 0, nil
}
