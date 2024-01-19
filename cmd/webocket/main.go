/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	mapValue := map[string]string{}
	err := json.Unmarshal([]byte("{}sfdbsdifsduf"), &mapValue)
	fmt.Print(err)
}
