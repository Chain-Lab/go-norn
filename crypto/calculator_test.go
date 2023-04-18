package crypto

import (
	"crypto/rand"
	"testing"
)

func TestCalculator(t *testing.T) {
	n, pp, err := GenerateParams()

	if err != nil {
		t.Fatal(err)
	}

	calculator := Calculator{
		order:      n,
		proofParam: pp,
		timeParam:  1000,
	}

	msg, err := rand.Prime(rand.Reader, 3)

	if err != nil {
		t.Fatal(err)
	}

	result, pi := calculator.calculate(msg)

	if !calculator.Verify(msg, pi, result) {
		t.Fatal("Verify failed.")
	}
}
