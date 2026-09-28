// Command repro 复现 btreeins 在较大规模下的已知异常行为。
//
//	go run ./repro
package main

import (
	"fmt"

	"btreeins"
)

func main() {
	const n = 200
	tr := btreeins.New(3)
	for k := n - 1; k >= 0; k-- {
		tr.Insert(k, "v")
	}

	missing := 0
	for k := 0; k < n; k++ {
		if _, ok := tr.Get(k); !ok {
			missing++
		}
	}
	fmt.Printf("倒序插入 %d 个键后，%d 个键取不回来\n", n, missing)

	small := btreeins.New(3)
	for _, k := range []int{10, 20, 30, 40} {
		small.Insert(k, "v")
	}
	fmt.Printf("Scan(20, 30) 返回 %d 个条目，期望 1 个\n", len(small.Scan(20, 30)))

	shrunk := btreeins.New(2)
	for k := 0; k <= 40; k++ {
		shrunk.Insert(k, "v")
	}
	before := shrunk.Height()
	for k := 1; k <= 40; k++ {
		shrunk.Delete(k)
	}
	fmt.Printf("删到只剩 1 个键后 Height = %d（删除前 %d），期望 1\n", shrunk.Height(), before)
}
