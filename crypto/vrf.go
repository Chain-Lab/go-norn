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
	log "github.com/sirupsen/logrus"
	"math/big"
)

func VRFCalculate(curve elliptic.Curve, prv *ecdsa.PrivateKey, msg []byte) ([]byte, *big.Int, *big.Int, error) {
	N := curve.Params().N

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
