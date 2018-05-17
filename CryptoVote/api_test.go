// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"testing"
	"fmt"

)

func TestBlocks_Less(t *testing.T) {
	ints := []int{1, 2, 3, 4, 5, 6}
	for _, v := range ints {
		if v % 2 == 0 {
			ints = ints[1:]
		}
	}
	fmt.Println(ints)
}

