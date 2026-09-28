# btreeins

B+ 树有序索引库：整数键唯一且有序，字符串值随键落在叶子层；内部节点保存分隔键与孩子指针，
最小度数 `t` 决定节点容量（最多 `2t-1` 个键，非根节点最少 `t-1` 个键）。

- 语言/依赖：Go 1.24，仅标准库，无第三方依赖。
- 构建：`bash scripts/build.sh`（即 `go build ./...`）。
- 自检：`bash scripts/check.sh`（`check/` 是**固定验收程序，勿改**）。
- 既有用例：`go test ./...`。
- 复现脚本：`go run ./repro`。

## 用法

```go
tr := btreeins.New(3)
tr.Insert(10, "a")
tr.Insert(20, "b")

if v, ok := tr.Get(10); ok { /* v == "a" */ }
tr.Scan(10, 20) // 落在 [10, 20) 内的条目，按键升序
tr.Delete(10)
tr.Len()    // 键值对数量
tr.Height() // 层数，只有一个叶子时为 1
```

## 对外契约

1. **键唯一且有序**：重复键插入不生效（`Insert` 返回 `false`，计数不变）；键在叶子层按键升序排列。
2. **分裂后所有叶子同深、键序正确**：任意插入顺序下，各叶子到根的边数一致，
   且按键升序遍历得到的序列有序、无重复、无遗漏。
3. **删除后借位/合并维持最小度数 `t`**：每个非根节点键数不少于 `t-1`；
   可选地借位或与兄弟合并，不得留下违反最小度数的节点。
4. **范围查询半开区间 `[lo, hi)`、升序**：`Scan` 只返回 `lo <= key < hi` 的条目，按键升序；
   `lo >= hi` 时返回空。
5. **根分裂与根塌缩正确**：根溢出时提升新根并使高度加一；根键数降为 0 且有唯一孩子时，
   该孩子成为新根，高度减一。
6. **迭代顺序确定、与插入顺序无关**：同一键集合不论以何种顺序插入，遍历结果逐元素相同。
7. **同一操作序列重放结果逐元素相同**：不依赖 map 迭代顺序、不依赖时间。
8. **空树/单元素树边界**：空树 `Len()==0`、`Height()==1`、`Get`/`Scan` 无结果；
   单元素树删除后回到空树。
9. **`repro.go`（复现脚本，跑得起来）能复现**当前异常行为。

## 诊断接口

`Diagnostics()` 返回**只读**结构快照，供自检与固定件核对不变量：

```go
type Diagnostics struct {
    Height         int   // 层数（只有一个叶子时为 1）
    LeafDepths     []int // 每个叶子到根的边数，升序
    MinNonRootKeys int   // 非根节点里最少的键数（无则 -1）
    Size           int   // 键值对数量
}
```

`New`、`Len`、`Height`、`Get`、`Insert`、`Delete`、`Scan`、`Diagnostics` 的对外签名已定死，不要改。

## 目录

```
btree.go         库实现
btree_test.go    既有用例
repro/repro.go   复现脚本（go run ./repro）
check/main.go    固定验收程序（8 个场景，勿改）
scripts/build.sh 构建
scripts/check.sh 自检入口
```
