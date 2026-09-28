package btreeins

import (
	"reflect"
	"strconv"
	"testing"
)

func keysOf(entries []Entry) []int {
	out := make([]int, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Key)
	}
	return out
}

func TestInsertAndGetSmall(t *testing.T) {
	tr := New(3)
	for _, k := range []int{10, 20, 30, 40} {
		if !tr.Insert(k, "v"+strconv.Itoa(k)) {
			t.Fatalf("Insert(%d) 返回 false", k)
		}
	}

	if tr.Len() != 4 {
		t.Fatalf("Len = %d, 期望 4", tr.Len())
	}
	if tr.Height() != 1 {
		t.Fatalf("Height = %d, 期望 1", tr.Height())
	}
	if v, ok := tr.Get(30); !ok || v != "v30" {
		t.Fatalf("Get(30) = %q,%v, 期望 v30,true", v, ok)
	}
	if _, ok := tr.Get(50); ok {
		t.Fatalf("Get(50) 应返回 false")
	}
}

func TestScanHalfOpenSmall(t *testing.T) {
	tr := New(3)
	for _, k := range []int{10, 20, 30, 40} {
		tr.Insert(k, "v"+strconv.Itoa(k))
	}

	if got := keysOf(tr.Scan(15, 35)); !reflect.DeepEqual(got, []int{20, 30}) {
		t.Fatalf("Scan(15,35) = %v, 期望 [20 30]", got)
	}
	if got := keysOf(tr.Scan(0, 100)); !reflect.DeepEqual(got, []int{10, 20, 30, 40}) {
		t.Fatalf("Scan(0,100) = %v, 期望 [10 20 30 40]", got)
	}
}

func TestDeleteSmall(t *testing.T) {
	tr := New(3)
	for _, k := range []int{10, 20, 30, 40} {
		tr.Insert(k, "v"+strconv.Itoa(k))
	}

	if !tr.Delete(20) {
		t.Fatalf("Delete(20) 返回 false")
	}
	if _, ok := tr.Get(20); ok {
		t.Fatalf("Get(20) 应返回 false")
	}
	if tr.Len() != 3 {
		t.Fatalf("Len = %d, 期望 3", tr.Len())
	}
	if got := keysOf(tr.Scan(0, 100)); !reflect.DeepEqual(got, []int{10, 30, 40}) {
		t.Fatalf("Scan(0,100) = %v, 期望 [10 30 40]", got)
	}
	if tr.Delete(99) {
		t.Fatalf("Delete(99) 应返回 false")
	}
}

func TestInsertOrderIndependent(t *testing.T) {
	tr := New(3)
	for _, k := range []int{40, 10, 30, 20} {
		tr.Insert(k, "v"+strconv.Itoa(k))
	}

	if got := keysOf(tr.Scan(0, 100)); !reflect.DeepEqual(got, []int{10, 20, 30, 40}) {
		t.Fatalf("Scan(0,100) = %v, 期望 [10 20 30 40]", got)
	}
}

func TestEmptyAndSingle(t *testing.T) {
	tr := New(3)
	if tr.Len() != 0 {
		t.Fatalf("空树 Len = %d, 期望 0", tr.Len())
	}
	if _, ok := tr.Get(1); ok {
		t.Fatalf("空树 Get 应返回 false")
	}
	if got := tr.Scan(0, 100); len(got) != 0 {
		t.Fatalf("空树 Scan = %v, 期望空", got)
	}

	tr.Insert(5, "v5")
	if tr.Len() != 1 {
		t.Fatalf("单元素 Len = %d, 期望 1", tr.Len())
	}
	if v, ok := tr.Get(5); !ok || v != "v5" {
		t.Fatalf("Get(5) = %q,%v, 期望 v5,true", v, ok)
	}
	if !tr.Delete(5) || tr.Len() != 0 {
		t.Fatalf("删除唯一元素后 Len = %d, 期望 0", tr.Len())
	}
}
