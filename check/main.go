// Command check 是 GSB132 btreeins 的固定验收程序。
//
// 用法：
//
//	go run ./check            跑全部场景
//	go run ./check -list      列出场景名
//	go run ./check --only insert  只跑某一组
//
// 输出逐场景 `PASS <组>/<名>` 或 `FAIL <组>/<名>  期望=… 实际=…`，
// 结尾 `结果：通过 x/N`；全过 exit 0，否则 exit 1。失败不早退。
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"btreeins"
)

type assertionError struct {
	expected string
	actual   string
}

func (e *assertionError) Error() string { return "assertion failed" }

func showInts(v []int) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = fmt.Sprint(x)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func keysOf(entries []btreeins.Entry) []int {
	out := make([]int, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Key)
	}
	return out
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func expectInt(expected, actual int) error {
	if expected != actual {
		return &assertionError{fmt.Sprint(expected), fmt.Sprint(actual)}
	}
	return nil
}

func expectInts(expected, actual []int) error {
	if !sameInts(expected, actual) {
		return &assertionError{showInts(expected), showInts(actual)}
	}
	return nil
}

func expectAllColumns(tr *btreeins.Tree, lo, hi int) error {
	for k := lo; k < hi; k++ {
		if _, ok := tr.Get(k); !ok {
			return &assertionError{fmt.Sprintf("Get(%d) 命中", k), "未命中"}
		}
	}
	return nil
}

type scenario struct {
	group string
	name  string
	run   func() error
}

func scenarios() []scenario {
	return []scenario{
		{
			group: "insert",
			name:  "duplicate-rejected",
			run: func() error {
				tr := btreeins.New(3)
				if !tr.Insert(5, "a") {
					return &assertionError{`Insert(5,"a") = true`, "false"}
				}
				if tr.Insert(5, "b") {
					return &assertionError{`Insert(5,"b") = false`, "true"}
				}
				if err := expectInt(1, tr.Len()); err != nil {
					return err
				}
				v, ok := tr.Get(5)
				if !ok || v != "a" {
					return &assertionError{`Get(5) = "a",true`, fmt.Sprintf("Get(5) = %q,%v", v, ok)}
				}
				return nil
			},
		},
		{
			group: "insert",
			name:  "reverse-order-get-all",
			run: func() error {
				const n = 300
				tr := btreeins.New(3)
				for k := n - 1; k >= 0; k-- {
					if !tr.Insert(k, fmt.Sprintf("v%d", k)) {
						return &assertionError{fmt.Sprintf("Insert(%d) = true", k), "false"}
					}
				}
				if err := expectInt(n, tr.Len()); err != nil {
					return err
				}
				for k := 0; k < n; k++ {
					v, ok := tr.Get(k)
					if !ok {
						return &assertionError{fmt.Sprintf("Get(%d) 命中", k), "未命中"}
					}
					if want := fmt.Sprintf("v%d", k); v != want {
						return &assertionError{fmt.Sprintf("Get(%d) = %q", k, want), fmt.Sprintf("%q", v)}
					}
				}
				return nil
			},
		},
		{
			group: "split",
			name:  "same-depth-leaves",
			run: func() error {
				const n = 512
				tr := btreeins.New(4)
				for k := n - 1; k >= 0; k-- {
					tr.Insert(k, "v")
				}
				d := tr.Diagnostics()
				if err := expectInt(n, d.Size); err != nil {
					return err
				}
				if len(d.LeafDepths) < 2 {
					return &assertionError{"分裂后叶子数 >= 2", fmt.Sprint(len(d.LeafDepths))}
				}
				for _, depth := range d.LeafDepths {
					if depth != d.LeafDepths[0] {
						return &assertionError{"所有叶子同深 " + fmt.Sprint(d.LeafDepths[0]), showInts(d.LeafDepths)}
					}
				}
				if err := expectInt(d.Height-1, d.LeafDepths[0]); err != nil {
					return err
				}
				if d.MinNonRootKeys < 3 {
					return &assertionError{"非根节点最少键数 >= 3", fmt.Sprint(d.MinNonRootKeys)}
				}
				return expectAllColumns(tr, 0, n)
			},
		},
		{
			group: "split",
			name:  "interleaved-order",
			run: func() error {
				const n = 200
				tr := btreeins.New(3)
				for i := 0; i < n; i++ {
					tr.Insert((i*37+11)%n, "v")
				}
				if err := expectAllColumns(tr, 0, n); err != nil {
					return err
				}
				want := make([]int, 0, n)
				for k := 0; k < n; k++ {
					want = append(want, k)
				}
				return expectInts(want, keysOf(tr.Scan(0, n)))
			},
		},
		{
			group: "delete",
			name:  "underflow-merged",
			run: func() error {
				const n = 400
				tr := btreeins.New(3)
				for k := 0; k < n; k++ {
					tr.Insert(k, "v")
				}
				for k := 0; k < 340; k++ {
					if !tr.Delete(k) {
						return &assertionError{fmt.Sprintf("Delete(%d) = true", k), "false"}
					}
				}
				d := tr.Diagnostics()
				if err := expectInt(60, d.Size); err != nil {
					return err
				}
				if d.MinNonRootKeys < 2 {
					return &assertionError{"非根节点最少键数 >= 2", fmt.Sprint(d.MinNonRootKeys)}
				}
				return expectAllColumns(tr, 340, n)
			},
		},
		{
			group: "delete",
			name:  "height-after-collapse",
			run: func() error {
				tr := btreeins.New(2)
				for k := 0; k <= 40; k++ {
					tr.Insert(k, "v")
				}
				if tr.Height() < 2 {
					return &assertionError{"插入 41 个键后 Height >= 2", fmt.Sprint(tr.Height())}
				}
				for k := 1; k <= 40; k++ {
					if !tr.Delete(k) {
						return &assertionError{fmt.Sprintf("Delete(%d) = true", k), "false"}
					}
				}
				if err := expectInt(1, tr.Len()); err != nil {
					return err
				}
				if _, ok := tr.Get(0); !ok {
					return &assertionError{"Get(0) 命中", "未命中"}
				}
				return expectInt(1, tr.Height())
			},
		},
		{
			group: "scan",
			name:  "half-open-range",
			run: func() error {
				tr := btreeins.New(3)
				for _, k := range []int{10, 20, 30, 40} {
					tr.Insert(k, "v")
				}
				if err := expectInts([]int{20}, keysOf(tr.Scan(20, 30))); err != nil {
					return err
				}
				if got := keysOf(tr.Scan(10, 10)); len(got) != 0 {
					return &assertionError{"Scan(10,10) 为空", showInts(got)}
				}
				if got := keysOf(tr.Scan(35, 10)); len(got) != 0 {
					return &assertionError{"Scan(35,10) 为空", showInts(got)}
				}
				return nil
			},
		},
		{
			group: "scan",
			name:  "ascending-deterministic",
			run: func() error {
				const n = 200
				want := make([]int, 0, n-1)
				for k := 0; k < n-1; k++ {
					want = append(want, k)
				}

				a := btreeins.New(3)
				for k := 0; k < n; k++ {
					a.Insert(k, "v")
				}
				b := btreeins.New(3)
				for k := n - 1; k >= 0; k-- {
					b.Insert(k, "v")
				}

				ga := keysOf(a.Scan(0, n-1))
				if err := expectInts(want, ga); err != nil {
					return err
				}
				if err := expectInts(want, keysOf(b.Scan(0, n-1))); err != nil {
					return err
				}
				return expectInts(ga, keysOf(a.Scan(0, n-1)))
			},
		},
	}
}

func runScenario(run func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return run()
}

func main() {
	var only string
	list := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-list" || args[i] == "--list":
			list = true
		case args[i] == "--only":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--only 缺少取值")
				os.Exit(2)
			}
			i++
			only = args[i]
		case strings.HasPrefix(args[i], "--only="):
			only = strings.TrimPrefix(args[i], "--only=")
		default:
			fmt.Fprintln(os.Stderr, "无法识别的参数："+args[i])
			os.Exit(2)
		}
	}

	all := scenarios()

	if list {
		for _, s := range all {
			fmt.Println(s.group + "/" + s.name)
		}
		os.Exit(0)
	}

	passed, ran := 0, 0
	for _, s := range all {
		if only != "" && s.group != only {
			continue
		}
		ran++
		label := s.group + "/" + s.name

		err := runScenario(s.run)
		if err == nil {
			passed++
			fmt.Println("PASS " + label)
			continue
		}

		expected, actual := "(未抛断言)", err.Error()
		var ae *assertionError
		if errors.As(err, &ae) {
			expected, actual = ae.expected, ae.actual
		}
		fmt.Printf("FAIL %s  期望=%s 实际=%s\n", label, expected, actual)
	}

	fmt.Printf("结果：通过 %d/%d\n", passed, ran)
	if passed != ran {
		os.Exit(1)
	}
}
