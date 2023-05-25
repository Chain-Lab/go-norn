/**
  @author: decision
  @date: 2023/3/?
  @note: VDF 计算协程，从 channel 中接收任务，计算后再放入 channel
**/

package crypto

import (
	"crypto/rand"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"math/big"
	"sync"
)

var (
	calculatorOnce sync.Once
	calculatorInst *Calculator
)

type Calculator struct {
	// 接收传入的新参数
	seedChannel      chan *big.Int
	prevProofChannel chan *big.Int
	// 传出计算结果
	resultChannel chan *big.Int
	proofChannel  chan *big.Int

	// 这三个参数区块存储在创世区块上
	proofParam *big.Int
	order      *big.Int
	timeParam  int64

	seed  *big.Int
	proof *big.Int

	changed    bool
	changeLock sync.RWMutex
}

// GetCalculatorInstance 所有的对 VDF 计算类的操作必须从这里获取实例
func GetCalculatorInstance() *Calculator {
	if calculatorInst == nil {
		log.Fatal("Calculator not init.")
		return nil
	}
	return calculatorInst
}

func CalculatorInitialization(pp *big.Int, order *big.Int, t int64) {
	calculatorOnce.Do(func() {
		calculatorInst = &Calculator{
			seedChannel:      make(chan *big.Int),
			prevProofChannel: make(chan *big.Int),
			resultChannel:    make(chan *big.Int),
			proofChannel:     make(chan *big.Int),

			proofParam: pp,
			order:      order,
			timeParam:  t,

			changed: false,
		}

		go calculatorInst.run()
	})
}

// GetSeedParams 读取计算信息，如果channel中有数据则优先获取
func (c *Calculator) GetSeedParams() (*big.Int, *big.Int) {
	seed := new(big.Int)
	proof := new(big.Int)

	if len(c.resultChannel) != 0 {
		seed = <-c.resultChannel
		proof = <-c.proofChannel
		return seed, proof
	}

	c.changeLock.RLock()
	seed.Set(c.seed)
	seed.Set(c.proof)
	c.changeLock.RUnlock()

	return seed, proof
}

// AppendNewSeed 在计算运行时修改此时的运行参数
func (c *Calculator) AppendNewSeed(seed *big.Int, proof *big.Int) {
	c.changeLock.RLock()

	// 检查如果当前的 seed 没有变化就直接返回
	if c.seed.Cmp(seed) == 0 {
		c.changeLock.RUnlock()

		return
	}

	c.changeLock.RUnlock()

	c.seedChannel <- seed
	c.prevProofChannel <- proof
	c.changed = true
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

func (c *Calculator) run() {
	for {
		select {
		case seed := <-c.seedChannel:
			c.changeLock.Lock()

			c.seed = seed
			c.proof = <-c.prevProofChannel

			c.changeLock.Unlock()
			pi, result := c.calculate(seed)

			if pi != nil {
				c.resultChannel <- result
				c.proofChannel <- pi
			}
		}
	}
}

// calculate 输入参数 seed，根据对象的参数来计算 VDF
func (c *Calculator) calculate(seed *big.Int) (*big.Int, *big.Int) {
	pi, r := big.NewInt(1), big.NewInt(1)
	g := new(big.Int)
	result := new(big.Int)
	result.Set(seed)

	// g = seed
	g.Set(seed)

	// result = g^(2^t)
	// 注： 这里如果直接计算 m^a mod n 的时间复杂度是接近 O(t) 的
	// 所以在 t 足够大的情况下是可以抵御攻击的
	for round := int64(0); round < c.timeParam; round++ {
		if c.changed {
			return nil, nil
		}

		// result = result^2 mod n
		result = result.Mul(result, result)
		result = result.Mod(result, c.order)

		// tmp = 2 * r
		tmp := big.NewInt(2)
		tmp = tmp.Mul(tmp, r)

		b := tmp.Div(tmp, c.proofParam) // b = tmp / l
		r = tmp.Mod(tmp, c.proofParam)  // r = tmp * l

		// pi = g^b * pi^2 mod n
		pi := pi.Mul(pi, pi)
		tmp = g.Exp(g, b, c.order)
		pi = pi.Mul(pi, tmp)
		pi = pi.Mod(pi, c.order)
	}

	return result, pi
}

// Verify 验证 VDF 计算结果 result == pi^l * seed^s
// 具体细节见论文 - Simple Verifiable Delay Functions
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

func GenerateGenesisParams() (*common.GenesisParams, error) {
	genesisParams := new(common.GenesisParams)
	order, pp, err := GenerateParams()

	if err != nil {
		return nil, err
	}

	seed, err := rand.Prime(rand.Reader, 256)

	if err != nil {
		return nil, err
	}

	genesisParams.Order = [256]byte(order.Bytes())
	genesisParams.TimeParam = 10000000
	genesisParams.VerifyParam = [32]byte(pp.Bytes())
	genesisParams.Seed = [32]byte(seed.Bytes())

	return genesisParams, nil
}
