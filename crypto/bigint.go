/**
  @author: decision
  @date: 2023/5/24
  @note:
**/

package crypto

import "math/big"

// BigPow 返回 a^b
func BigPow(a, b int64) *big.Int {
	r := big.NewInt(a)
	return r.Exp(r, big.NewInt(b), nil)
}
