/**
  @author: decision
  @date: 2023/4/23
  @note:
**/

package crypto

import (
	"crypto/elliptic"
	"encoding/hex"
	"testing"
)

func TestVRFCalculateAndVerify(t *testing.T) {
	//prv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	//if err != nil {
	//	t.Fatal(err)
	//}

	msg := []byte("test-vrf")
	randomNumber, s, t1, err := VRFCalculate(elliptic.P256(), msg)
	if err != nil {
		t.Fatal(err)
	}

	bytesKey, _ := hex.DecodeString("02494712e50a07523e9baa89367fbea76b4a968073784859383593154b287c3b48")
	pubKey := Bytes2PublicKey(bytesKey)

	result, err := VRFVerify(elliptic.P256(), pubKey, msg, s, t1, randomNumber)
	if err != nil {
		t.Fatal(err)
	}

	if !result {
		t.Fatal("Verify random failed.")
	}
}
