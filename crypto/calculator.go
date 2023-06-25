/**
  @author: decision
  @date: 2023/3/?
  @note: VDF 计算协程，从 channel 中接收任务，计算后再放入 channel
**/

package crypto

import (
	"crypto/rand"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"math/big"
	"sync"
)

var (
	calculatorOnce sync.Once                   // 单例，实例化一次 calculator
	calculatorInst *Calculator                 // calculator 的实例
	zero           *big.Int    = big.NewInt(0) // 常量，大整数下的 0
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
			// 传递消息的管道，外界 -> calculator
			seedChannel:      make(chan *big.Int, 1),
			prevProofChannel: make(chan *big.Int, 1),
			// 传递消息的管道， calculator -> 外界
			resultChannel: make(chan *big.Int, 1),
			proofChannel:  make(chan *big.Int, 1),

			// 证明参数，阶数，时间参数
			proofParam: pp,
			order:      order,
			timeParam:  t,

			// 当前阶段的输出和证明
			seed:  big.NewInt(0),
			proof: big.NewInt(0),

			// 输入是否出现变化（是否需要直接进入下一轮）
			changed: false,
		}

		go calculatorInst.run()
	})
}

// GetSeedParams 读取计算信息，如果channel中有数据则优先获取
func (c *Calculator) GetSeedParams() (*big.Int, *big.Int) {
	seed := new(big.Int)
	proof := new(big.Int)

	log.Debugf("Channel length: %d", len(c.resultChannel))
	if len(c.resultChannel) != 0 {
		seed = <-c.resultChannel
		proof = <-c.proofChannel
		return seed, proof
	}

	c.changeLock.RLock()
	seed.Set(c.seed)
	proof.Set(c.proof)
	c.changeLock.RUnlock()

	return seed, proof
}

// AppendNewSeed 在计算运行时修改此时的运行参数
func (c *Calculator) AppendNewSeed(seed *big.Int, proof *big.Int) {
	c.changeLock.Lock()
	log.Infof("Trying append new seed %s.", hex.EncodeToString(seed.Bytes()))

	// 检查如果当前的 seed 没有变化就直接返回 或者
	// 如果当前的 seed 不是初始的0，并且输入无法通过验证则不更新
	if c.seed.Cmp(seed) == 0 || (c.seed.Cmp(zero) != 0 && !c.Verify(c.seed, proof, seed)) {
		c.changeLock.Unlock()
		return
	}

	// todo： 这里切换的地方感觉还是存在问题
	c.changed = true
	c.changeLock.Unlock()

	c.seedChannel <- seed
	c.prevProofChannel <- proof
}

// GenerateParams 用于生成计算参数，返回 order(n), proof_param
func GenerateParams() (*big.Int, *big.Int, error) {
	r := rand.Reader
	n := new(big.Int)

	// 随机获取 512 bits 的素数
	p, err := rand.Prime(r, 512)
	if err != nil {
		return nil, nil, err
	}

	q, err := rand.Prime(r, 512)
	if err != nil {
		return nil, nil, err
	}

	// n = p * q
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
			log.Debugf("Start new VDF calculate.")

			c.changed = false
			c.changeLock.Unlock()
			c.seed = seed
			c.proof = <-c.prevProofChannel

			result, pi := c.calculate(seed)

			if pi != nil {
				log.Debugf("Calculate result: %s", hex.EncodeToString(result.Bytes()))
				log.Debugf("Calculate proof: %s", hex.EncodeToString(pi.Bytes()))
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

	// result = g^(2^t)
	// 注： 这里如果直接计算 m^a mod n 的时间复杂度是接近 O(t) 的
	// 所以在 t 足够大的情况下是可以抵御攻击的
	for round := int64(0); round < c.timeParam; round++ {
		//log.Infoln(round)
		if c.changed {
			return nil, nil
		}

		// result = result^2 mod n
		result.Mul(result, result)
		result.Mod(result, c.order)

		// tmp = 2 * r
		tmp := big.NewInt(2)
		b := new(big.Int)

		tmp.Mul(tmp, r)

		b.Div(tmp, c.proofParam) // b = tmp / l
		r.Mod(tmp, c.proofParam) // r = tmp % l
		// pi = g^b * pi^2 mod n
		pi.Mul(pi, pi)
		g.Exp(seed, b, c.order)
		pi.Mul(pi, g)
		pi.Mod(pi, c.order)
	}

	return result, pi
}

// Verify 验证 VDF 计算结果 result == pi^l * seed^s
// 具体细节见论文 - Simple Verifiable Delay Functions
func (c *Calculator) Verify(seed *big.Int, pi *big.Int, result *big.Int) bool {
	// todo: 写一下 calculate 和 verify 实现的公式的过程
	r := big.NewInt(2)
	t := big.NewInt(int64(c.timeParam))

	// r = r^t mod pp
	r = r.Exp(r, t, c.proofParam)

	// h = pi^pp
	h := pi.Exp(pi, c.proofParam, c.order)
	// s = seed
	s := seed.Exp(seed, r, c.order)

	h = h.Mul(h, s)
	h = h.Mod(h, c.order)

	return result.Cmp(h) == 0
}

// GenerateGenesisParams 打包 VDF 计算参数
func GenerateGenesisParams() (*common.GenesisParams, error) {
	genesisParams := new(common.GenesisParams)
	// 获取参数：群的阶数以及证明参数
	order, pp, err := GenerateParams()

	if err != nil {
		return nil, err
	}

	seed, err := rand.Prime(rand.Reader, 256)

	if err != nil {
		return nil, err
	}

	genesisParams.Order = [128]byte(order.Bytes())
	genesisParams.TimeParam = 1000000000
	//genesisParams.TimeParam = 1000
	genesisParams.VerifyParam = [32]byte(pp.Bytes())
	genesisParams.Seed = [32]byte(seed.Bytes())

	return genesisParams, nil
}
