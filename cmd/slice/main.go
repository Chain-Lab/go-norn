/**
  @author: decision
  @date: 2023/9/15
  @note:
**/

package main

func removePrefixIfExists(hexString string) string {
	if hexString[:2] == "0x" {
		return hexString[2:]
	} else {
		return hexString
	}
}

func main() {
	txHash := "0x0e30221f8c7181e625fa821deeae8ff4261ceb2e1e68c5e0a469e9cad3b1f495"
	println(removePrefixIfExists(txHash))
}
