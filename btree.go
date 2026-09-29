// Package btreeins 提供一棵 B+ 树实现的有序索引。
//
// 键是整数、唯一且有序；值是字符串。叶子层按序排列，内部节点保存
// 分隔键与孩子指针；最小度数 t 决定每个节点的容量（最多 2t-1 个键，
// 非根节点最少 t-1 个键）。
package btreeins

import "sort"

// Entry 是一个键值对。
type Entry struct {
	Key   int
	Value string
}

// Diagnostics 是结构快照，供自检与固定件核对不变量。
type Diagnostics struct {
	Height         int   // 层数（只有一个叶子时为 1）
	LeafDepths     []int // 每个叶子到根的边数，升序
	MinNonRootKeys int   // 非根节点里最少的键数（无则 -1）
	Size           int   // 键值对数量
}

type node struct {
	leaf     bool
	keys     []int
	vals     []string
	children []*node
}

// Tree 是一棵 B+ 树。
type Tree struct {
	t      int
	root   *node
	height int
	size   int
}

// New 新建一棵最小度数为 t 的空树。
func New(t int) *Tree {
	if t < 2 {
		panic("btreeins: t 至少为 2")
	}
	return &Tree{t: t, root: &node{leaf: true}, height: 1}
}

// Len 返回键值对数量。
func (tr *Tree) Len() int { return tr.size }

// Height 返回层数（只有一个叶子时为 1）。
func (tr *Tree) Height() int { return tr.height }

// Get 按键取值。
func (tr *Tree) Get(key int) (string, bool) {
	n := tr.root
	for !n.leaf {
		n = n.children[upperBound(n.keys, key)]
	}
	i := sort.SearchInts(n.keys, key)
	if i < len(n.keys) && n.keys[i] == key {
		return n.vals[i], true
	}
	return "", false
}

// Insert 插入一个键值对。键已存在时返回 false。
func (tr *Tree) Insert(key int, value string) bool {
	inserted, pk, right, split := tr.insert(tr.root, key, value)
	if !inserted {
		return false
	}
	if split {
		tr.root = &node{keys: []int{pk}, children: []*node{tr.root, right}}
		tr.height++
	}
	tr.size++
	return true
}

// insert 递归插入。inserted 表示键是否为新键；split 为 true 时
// pk/right 是提升到父节点的分隔键与新建的右兄弟。
func (tr *Tree) insert(n *node, key int, value string) (inserted bool, pk int, right *node, split bool) {
	if n.leaf {
		i := sort.SearchInts(n.keys, key)
		if i < len(n.keys) && n.keys[i] == key {
			return false, 0, nil, false
		}
		n.keys = insertInt(n.keys, i, key)
		n.vals = insertStr(n.vals, i, value)
		if len(n.keys) <= 2*tr.t-1 {
			return true, 0, nil, false
		}
		pk, right := tr.splitLeaf(n)
		return true, pk, right, true
	}

	i := upperBound(n.keys, key)
	inserted, pk, right, split = tr.insert(n.children[i], key, value)
	if !inserted || !split {
		return inserted, 0, nil, false
	}

	n.keys = insertInt(n.keys, i, pk)
	n.children = insertNode(n.children, i+1, right)
	if len(n.keys) <= 2*tr.t-1 {
		return true, 0, nil, false
	}
	pk, right = tr.splitInternal(n)
	return true, pk, right, true
}

func (tr *Tree) splitLeaf(n *node) (int, *node) {
	mid := len(n.keys) / 2
	right := &node{
		leaf: true,
		keys: append([]int(nil), n.keys[mid:]...),
		vals: append([]string(nil), n.vals[mid:]...),
	}
	n.keys = n.keys[:mid]
	n.vals = n.vals[:mid]
	return right.keys[0], right
}

func (tr *Tree) splitInternal(n *node) (int, *node) {
	mid := len(n.keys) / 2
	pk := n.keys[mid]
	right := &node{
		keys:     append([]int(nil), n.keys[mid+1:]...),
		children: append([]*node(nil), n.children[mid+1:]...),
	}
	n.keys = n.keys[:mid]
	n.children = n.children[:mid+1]
	return pk, right
}

// Delete 删除一个键。键不存在时返回 false。
func (tr *Tree) Delete(key int) bool {
	if !tr.delete(tr.root, key) {
		return false
	}
	tr.size--
	for !tr.root.leaf && len(tr.root.keys) == 0 {
		tr.root = tr.root.children[0]
		tr.height--
	}
	return true
}

func (tr *Tree) delete(n *node, key int) bool {
	if n.leaf {
		i := sort.SearchInts(n.keys, key)
		if i >= len(n.keys) || n.keys[i] != key {
			return false
		}
		n.keys = append(n.keys[:i], n.keys[i+1:]...)
		n.vals = append(n.vals[:i], n.vals[i+1:]...)
		return true
	}

	i := upperBound(n.keys, key)
	if !tr.delete(n.children[i], key) {
		return false
	}
	if len(n.children[i].keys) < tr.t-1 {
		tr.rebalance(n, i)
	}
	return true
}

// rebalance 处理内部节点 n 的第 i 个孩子发生的下溢：先尝试向
// 左右兄弟借位，借不到就与兄弟合并。叶子借位/合并会同步维护分隔键。
func (tr *Tree) rebalance(n *node, i int) {
	child := n.children[i]

	if i > 0 && len(n.children[i-1].keys) > tr.t-1 {
		left := n.children[i-1]
		if child.leaf {
			k := left.keys[len(left.keys)-1]
			v := left.vals[len(left.vals)-1]
			left.keys = left.keys[:len(left.keys)-1]
			left.vals = left.vals[:len(left.vals)-1]
			child.keys = append([]int{k}, child.keys...)
			child.vals = append([]string{v}, child.vals...)
			n.keys[i-1] = child.keys[0]
		} else {
			child.keys = append([]int{n.keys[i-1]}, child.keys...)
			child.children = append([]*node{left.children[len(left.children)-1]}, child.children...)
			n.keys[i-1] = left.keys[len(left.keys)-1]
			left.keys = left.keys[:len(left.keys)-1]
			left.children = left.children[:len(left.children)-1]
		}
		return
	}

	if i < len(n.children)-1 && len(n.children[i+1].keys) > tr.t-1 {
		right := n.children[i+1]
		if child.leaf {
			k := right.keys[0]
			v := right.vals[0]
			right.keys = right.keys[1:]
			right.vals = right.vals[1:]
			child.keys = append(child.keys, k)
			child.vals = append(child.vals, v)
			n.keys[i] = right.keys[0]
		} else {
			child.keys = append(child.keys, n.keys[i])
			child.children = append(child.children, right.children[0])
			n.keys[i] = right.keys[0]
			right.keys = right.keys[1:]
			right.children = right.children[1:]
		}
		return
	}

	if i > 0 {
		tr.merge(n, i-1)
	} else {
		tr.merge(n, 0)
	}
}

// merge 把 n 的第 i 个孩子与其右兄弟（第 i+1 个）合并，
// 同时移除 n.keys[i]。
func (tr *Tree) merge(n *node, i int) {
	left := n.children[i]
	right := n.children[i+1]

	if left.leaf {
		left.keys = append(left.keys, right.keys...)
		left.vals = append(left.vals, right.vals...)
	} else {
		left.keys = append(left.keys, n.keys[i])
		left.keys = append(left.keys, right.keys...)
		left.children = append(left.children, right.children...)
	}

	n.keys = append(n.keys[:i], n.keys[i+1:]...)
	n.children = append(n.children[:i+1], n.children[i+2:]...)
}

// Scan 返回键落在 [lo, hi) 内的条目，按键升序。
func (tr *Tree) Scan(lo, hi int) []Entry {
	if lo >= hi {
		return nil
	}
	var out []Entry
	var walk func(n *node)
	walk = func(n *node) {
		if n.leaf {
			for i, k := range n.keys {
				if k >= lo && k < hi {
					out = append(out, Entry{Key: k, Value: n.vals[i]})
				}
			}
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(tr.root)
	return out
}

// Diagnostics 返回当前结构的快照。
func (tr *Tree) Diagnostics() Diagnostics {
	d := Diagnostics{Height: tr.height, Size: tr.size, MinNonRootKeys: -1}

	var walk func(n *node, depth int)
	walk = func(n *node, depth int) {
		if n != tr.root {
			if d.MinNonRootKeys < 0 || len(n.keys) < d.MinNonRootKeys {
				d.MinNonRootKeys = len(n.keys)
			}
		}
		if n.leaf {
			d.LeafDepths = append(d.LeafDepths, depth)
			return
		}
		for _, c := range n.children {
			walk(c, depth+1)
		}
	}
	walk(tr.root, 0)
	sort.Ints(d.LeafDepths)
	return d
}

func upperBound(a []int, x int) int {
	return sort.Search(len(a), func(i int) bool { return a[i] > x })
}

func insertInt(a []int, i, v int) []int {
	a = append(a, 0)
	copy(a[i+1:], a[i:])
	a[i] = v
	return a
}

func insertNode(a []*node, i int, v *node) []*node {
	a = append(a, nil)
	copy(a[i+1:], a[i:])
	a[i] = v
	return a
}

func insertStr(a []string, i int, v string) []string {
	a = append(a, "")
	copy(a[i+1:], a[i:])
	a[i] = v
	return a
}
