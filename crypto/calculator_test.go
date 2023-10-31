package crypto

import (
	"crypto/rand"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"testing"
)

func TestCalculator(t *testing.T) {
	n, pp, err := GenerateParams()
	tt260 := BigPow(2, 260)

	if err != nil {
		t.Fatal(err)
	}

	calculator := Calculator{
		order:      n,
		proofParam: pp,
		timeParam:  10000,
	}

	msg, err := rand.Prime(rand.Reader, 3)

	if err != nil {
		t.Fatal(err)
	}

	result, pi := calculator.calculate(msg)

	if !calculator.Verify(msg, pi, result) {
		t.Fatal("Verify failed.")
	}

	result.Cmp(tt260)
	log.Info(hex.EncodeToString(result.Bytes()))
}
