// Package crypto
// @Description: VDF 计算协程，从 channel 中接收任务，计算后再放入 channel
package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/metrics"
	log "github.com/sirupsen/logrus"
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

	prevSeed *big.Int
	seed     *big.Int
	proof    *big.Int

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

// CalculatorInitialization
//
//	@Description: 初始化 VDF 计算实例
//	@param pp - 证明所用到的安全参数，对应论文中的 l
//	@param order - 计算所在的群的阶
//	@param t - 计算的时间参数 t，需要计算的为 m^{2^{t}}
func CalculatorInitialization(pp *big.Int, order *big.Int, t int64) {
	calculatorOnce.Do(func() {
		calculatorInst = &Calculator{
			// 传递消息的管道，外界 -> calculator
			seedChannel:      make(chan *big.Int, 32),
			prevProofChannel: make(chan *big.Int, 32),
			// 传递消息的管道， calculator -> 外界
			resultChannel: make(chan *big.Int, 32),
			proofChannel:  make(chan *big.Int, 32),

			// 证明参数，阶数，时间参数
			proofParam: pp,
			order:      order,
			timeParam:  t,

			// 当前阶段的输出和证明
			prevSeed: big.NewInt(0),
			seed:     big.NewInt(0),
			proof:    big.NewInt(0),

			// 输入是否出现变化（是否需要直接进入下一轮）
			changed: false,
		}

		metrics.RoutineCreateCounterObserve(10)
		go calculatorInst.run()
		//go func(c *Calculator) {
		//	for {
		//		time.Sleep(100 * time.Millisecond)
		//		log.Infof("VDF Seed: %s", hex.EncodeToString(c.seed.Bytes()))
		//	}
		//}(calculatorInst)
	})
}

// GetSeedParams
//
//	@Description: 读取计算信息，如果channel中有数据则优先获取
//	@receiver c - 计算实例
//	@return *big.Int - 计算结果
//	@return *big.Int - 证明 pi
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
	seed.SetBytes(c.seed.Bytes())
	proof.SetBytes(c.proof.Bytes())
	c.changeLock.RUnlock()

	return seed, proof
}

// VerifyBlockVDF
//
//	@Description: 验证某个区块的 VDF 是否正确
//	@receiver c - 计算实例
//	@param seed - 传入的区块的 seed
//	@param proof - 证明参数
//	@return bool - 是否正确
func (c *Calculator) VerifyBlockVDF(seed *big.Int, proof *big.Int) bool {
	c.changeLock.Lock()
	defer c.changeLock.Unlock()

	// 如果区块的 seed 和上一个参数相同或 seed 与当前的 seed 相同，或者在不为 0 的情况下验证成功
	// 返回参数正确
	if c.prevSeed.Cmp(seed) == 0 || c.seed.Cmp(seed) == 0 || (c.seed.Cmp(
		zero) != 0 && c.
		Verify(c.
			seed,
			proof, seed)) {
		return true
	}

	return false
}

// AppendNewSeed
//
//	@Description: 在计算运行时修改此时的运行参数，常用语其它节点完成计算，但是本地还没完成计算
//	@receiver c
//	@param seed - 区块携带的 seed
//	@param proof - 证明 pi
func (c *Calculator) AppendNewSeed(seed *big.Int, proof *big.Int) {
	c.changeLock.Lock()
	defer c.changeLock.Unlock()

	log.Debugln("Now VDF seed: %s", hex.EncodeToString(c.seed.Bytes()))

	// 检查如果当前的 seed 没有变化就直接返回 或者
	// 如果当前的 seed 不是初始的0，并且输入无法通过验证则不更新
	if c.prevSeed.Cmp(seed) == 0 || c.seed.Cmp(seed) == 0 || (c.seed.Cmp(zero) != 0 && !c.Verify(c.seed, proof, seed)) {
		log.Debugln("Block VDF verify failed seed: %s, result: %s",
			hex.EncodeToString(c.seed.Bytes()), hex.EncodeToString(seed.
				Bytes()))
		return
	}

	c.changed = true
	log.Debugf("New Seed: %s, Proof: %s", hex.EncodeToString(seed.Bytes()),
		hex.EncodeToString(proof.Bytes()))
	c.seedChannel <- seed
	c.prevProofChannel <- proof
}

// GenerateParams
//
//	@Description: 用于生成计算参数，返回 order(n), proof_param
//	@return *big.Int - 阶
//	@return *big.Int - 证明参数
//	@return error
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

// run
//
//	@Description: VDF 计算运行主程序，如果其它节点传入了正确的参数，会中断当前计算并且开始新一轮的计算
//	@receiver c
func (c *Calculator) run() {
	for {
		select {
		case seed := <-c.seedChannel:
			c.changeLock.RLock()
			log.Infoln("Start new VDF calculate.")

			c.changed = false
			c.prevSeed.SetBytes(c.seed.Bytes())
			c.seed.SetBytes(seed.Bytes())
			log.Infof("Set seed to %s", hex.EncodeToString(c.seed.Bytes()))
			c.proof = <-c.prevProofChannel
			c.changeLock.RUnlock()

			result, pi := c.calculate(seed)

			if pi != nil {
				log.Debugf("Calculate result: %s", hex.EncodeToString(result.
					Bytes()))
				log.Debugf("Calculate proof: %s", hex.EncodeToString(pi.Bytes()))
				c.resultChannel <- result
				c.proofChannel <- pi
			}
		}
	}
}

// calculate
//
//	@Description: 输入参数 seed，根据对象的参数来计算 VDF，参考论文：A survey of two verifiable delay functions
//	@receiver c
//	@param seed - 当前轮计算的 seed
//	@return *big.Int - VDF 计算结果
//	@return *big.Int - VDF 证明 pi
func (c *Calculator) calculate(seed *big.Int) (*big.Int, *big.Int) {
	pi, r := big.NewInt(1), big.NewInt(1)
	g := new(big.Int)
	result := new(big.Int)
	result.SetBytes(seed.Bytes())

	// result = g^(2^t)
	// 注： 这里如果直接计算 m^a mod n 的时间复杂度是接近 O(t) 的
	// 所以在 t 足够大的情况下是可以抵御攻击的
	for round := int64(0); round < c.timeParam; round++ {
		//log.Infoln(round)
		if c.changed {
			log.Infoln("VDF seed changed")
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
	tSeed := big.NewInt(0)
	tSeed.SetBytes(seed.Bytes())
	tPi := big.NewInt(0)
	tPi.SetBytes(pi.Bytes())

	r := big.NewInt(2)
	t := big.NewInt(c.timeParam)

	// r = r^t mod pp
	r = r.Exp(r, t, c.proofParam)

	// h = pi^pp
	h := tPi.Exp(tPi, c.proofParam, c.order)
	// s = seed
	s := tSeed.Exp(tSeed, r, c.order)

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
	genesisParams.TimeParam = 10000000
	//genesisParams.TimeParam = 1000
	genesisParams.VerifyParam = [32]byte(pp.Bytes())
	genesisParams.Seed = [32]byte(seed.Bytes())

	return genesisParams, nil
}
