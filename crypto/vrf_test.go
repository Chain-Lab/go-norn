/**
  @author: decision
  @date: 2023/4/23
  @note:
**/

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestVRFCalculateAndVerify(t *testing.T) {
	prv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("test-vrf")
	randomNumber, s, t1, err := VRFCalculate(elliptic.P256(), msg)
	if err != nil {
		t.Fatal(err)
	}

	result, err := VRFVerify(elliptic.P256(), &prv.PublicKey, msg, s, t1, randomNumber)
	if err != nil {
		t.Fatal(err)
	}

	if !result {
		t.Fatal("Verify random failed.")
	}
}
