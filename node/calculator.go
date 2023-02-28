package node

import (
	"crypto/rand"
	"math/big"
)

type Calculator struct {
	seedChannel   chan *big.Int
	resultChannel chan *big.Int
	proofChannel  chan *big.Int

	// 这三个参数区块存储在创世区块上
	proofParam *big.Int
	order      *big.Int
	timeParam  int

	changed bool
}

func NewCalculator() (*Calculator, error) {
	return nil, nil
}

// GenerateParams 用于生成计算参数，返回 order(n), proof_param
func GenerateParams() (*big.Int, *big.Int, error) {
	r := rand.Reader
	n := new(big.Int)

	p, err := rand.Prime(r, 512)
	if err != nil {

		return nil, nil, err
	}

	q, err := rand.Prime(r, 512)
	if err != nil {
		return nil, nil, err
	}

	n = n.Mul(p, q)
	pp, err := rand.Prime(r, 256)

	if err != nil {
		return nil, nil, err
	}

	return n, pp, nil
}

func (c *Calculator) Run() {
	for {
		select {
		case seed := <-c.seedChannel:
			pi, result := c.calculate(seed)

			if pi != nil {
				c.resultChannel <- result
				c.proofChannel <- pi
			}
		}
	}
}

func (c *Calculator) calculate(seed *big.Int) (*big.Int, *big.Int) {
	pi, r := big.NewInt(1), big.NewInt(1)
	g := new(big.Int)
	result := new(big.Int)
	result.Set(seed)
	g.Set(seed)

	for round := 0; round < c.timeParam; round++ {
		if c.changed {
			return nil, nil
		}
		result = result.Mul(result, result)
		result = result.Mod(result, c.order)

		tmp := big.NewInt(2)
		tmp = tmp.Mul(tmp, r)
		b := tmp.Div(tmp, c.proofParam)
		r = tmp.Mod(tmp, c.proofParam)

		pi := pi.Mul(pi, pi)
		tmp = g.Exp(g, b, c.order)
		pi = pi.Mul(pi, tmp)
		pi = pi.Mod(pi, c.order)
	}

	return result, pi
}

func (c *Calculator) Verify(seed *big.Int, pi *big.Int, result *big.Int) bool {
	// todo: 写一下 calculate 和 verify 实现的公式的过程
	r := big.NewInt(2)
	t := big.NewInt(int64(c.timeParam))
	r = r.Exp(r, t, c.proofParam)

	h := pi.Exp(pi, c.proofParam, c.order)
	s := seed.Exp(seed, r, c.order)

	h = h.Mul(h, s)
	h = h.Mod(h, c.order)

	return result.Cmp(h) == 0
}
