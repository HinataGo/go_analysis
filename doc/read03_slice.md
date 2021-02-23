# 切片 （动态数组）
### 类型声明
```go
// T  即 type ， 多种类型，包括interface{}
[]T
```
- 片在编译期间的生成的类型只会包含切片中的元素类型， 这个由函数 cmd/compile/internal/types.NewSlice （
  go/src/cmd/compile/internal/types/type.go - 493行） 决定 
```go
func NewSlice(elem *Type) *Type {
	if t := elem.Cache.slice; t != nil {
		if t.Elem() != elem {
			Fatalf("elem mismatch")
		}
		return t
	}

	t := New(TSLICE)
	t.Extra = Slice{Elem: elem}
	elem.Cache.slice = t
	return t
}

```
- 返回结构体中的 Extra 字段是一个只包含切片内元素类型的结构 ————> t.Extra = Slice{Elem: elem}
- 切片元素类型在编译期间确定，编译器确定类型之后， 将类型存储在 Extra 字段 中，帮助程序在运行时动态获取。

## 1.数据结构
- 编译期切片是 cmd/compile/internal/types.Slice （type.go - 346 行） 类型决定
- 运行期间由 reflect.SliceHeader  （go/src/reflect/value.go  - 1994行）结构体表示 
```go
// Data 是指向数组的指针;
// Len 是当前切片的长度；
// Cap 是当前切片的容量，即 Data 数组的大小：    
type SliceHeader struct {
    Data uintptr
    Len  int
    Cap  int
}
```
- 理解切片： 一片连续的内存空间加上长度与容量的标识
- Data 
    - 一片连续的内存空间， 存储切片中的全部元素
    - 数组中的元素只是逻辑上的概念，底层存储其实都是连续的
    -data 的实际大小为cap ，存储的实际元素所占大小则是len
- 切片和数组密不可分：
    - 切片提供抽象概念， 我们通过切片对数组一部分连续片段进行引用，并且可以在运行期修改切片的长度（len） 和 范围（cap） 
    - 当底层数组len不足时， 会发生扩容操作，但是上层 切片看起来没有变化，底层指向 的 数组可能 变化。这就是我们只需要关注上层切片的操作即可
    
## 2.初始化
### 2.1 使用下标
- 三种初始化方式
    - 通过下标的方式获得 数组/ 切片 的一部分；
    - 使用字面量 []int{1, 2, 3} 初始化新的切片；
    - 使用关键字 make 创建切片：
```go
arr[0:3] / slice[0:3]
slice := []int{1, 2, 3}
slice := make([]int, 10)
```
- 下标创建切片，最接近汇编，也最原始。 是所有方法最底层的一种。编译器会将 下边创建转换成 OpSliceMake
- 这里使用 GOSSAFUNC=newSlice  go build slice.go 编译生成SSA文件
```go
func newSlice() []int {
  arr := [3]int{1, 2, 3}
  slice := arr[0:1]
  return slice
}
```
- 在decompose builtin 阶段  slice := arr[0:1] 对应语句如下：
```go
v27 (+6) = SliceMake <[]int> v11 v14 v17
...
name &arr[*[3]int]: v11
name slice.ptr[*int]: v11
name slice.len[int]: v14
name slice.cap[int]: v17 
```
- SliceMake 操作会接受四个参数创建新的切片，元素类型、数组指针、切片大小和容量
- 使用下标初始化切片不会拷贝原数组或者原切片中的数据，它只会创建一个指向原数组的切片结构体，所以修改新切片的数据也会修改原切片
### 2.2 字面量（字面量的方式创建切片，大部分的工作都会在编译期间完成）
- 字面量 []int{1, 2, 3}创建新切片 时，编译期间会被cmd/compile/internal/gc.slicelit 展开为如下代码，作用如下：
  - 1. 根据元素数量，推断底层数组大小，创建一个底层数组
  - 2. 将字面量元素 存储到初始化的 数组中
  - 3. 创建 一个同样 指向同样类型( [3]int ) 的数组指针
  - 4. 将静态存储区的数组vstat 赋值给vauto 指针所在的地址
  - 5. 通过[:] 操作获取一个底层使用vauto的切片 ---> 因此 [:]操作是最底层的一种创建方法
  ```go
  var vstat [3]int
  vstat[0] = 1
  vstat[1] = 2
  vstat[2] = 3
  var vauto *[3]int = new([3]int)
  *vauto = vstat
  slice := vauto[:]
  ```
### 2.3 关键字（ make 关键字创建切片时，很多工作都需要运行时的参与）
- 调用方必须向 make 函数传入切片的大小以及可选的容量
- 类型检查期间的 cmd/compile/internal/gc.typecheck1 函数会校验入参
```go
func typecheck1(n *Node, top int) (res *Node) {
  switch n.Op {
    ...
    case OMAKE:
    	args := n.List.Slice()
    
    i := 1
      switch t.Etype {
        case TSLICE:
          if i >= len(args) {
            yyerror("missing len argument to make(%v)", t)
            return n
          }
          
          l = args[i]
          i++
          var r *Node
          if i < len(args) {
            r = args[i]
          }
          ...
          if Isconst(l, CTINT) && r != nil && Isconst(r, CTINT) && l.Val().U.(*Mpint).Cmp(r.Val().U.(*Mpint)) > 0 {
            yyerror("len larger than cap in make(%v)", t)
            return n
          }
          
          n.Left = l
          n.Right = r
          n.Op = OMAKESLICE
      }
    ...
  }
}
```
- 这一过程会检查 
  - 1. len 必须传入
  - 2. cap >= len
  - 3. 当前函数会将OMAKE 节点转换成 OMAKESLICE
  - 4. 中间代码 生成的  cmd/compile/internal/gc.walkexpr 函数  根据下面两个条件 转换  OMAKESLICE的类型节点
    - 切片的大小和容量是否足够小；
    - 切片是否发生了逃逸，最终在堆上初始化
1. 不逃逸或者非常小 ，在栈上 或者 静态存储区创建数组并将 切片转换成 OpSliceMake 操作， 均在编译期间 完成。，直接转换成以下代码
```go
var arr [4]int
n := arr[:3]
```
2. 切片发生逃逸或者非常大时，运行时需要 runtime.makeslice 在堆上初始化切片
```go
func makeslice(et *_type, len, cap int) unsafe.Pointer {
	mem, overflow := math.MulUintptr(et.size, uintptr(cap))
	if overflow || mem > maxAlloc || len < 0 || len > cap {
		mem, overflow := math.MulUintptr(et.size, uintptr(len))
		if overflow || mem > maxAlloc || len < 0 {
			panicmakeslicelen()
		}
		panicmakeslicecap()
	}

	return mallocgc(mem, et, true)
}
// 计算切片占用内存空间 ，并在堆上 申请一片连续的内存
// 内存空间=切片中元素大小×切片容量
```
#### 错误检查
- 创建切片的过程中如果发生了以下错误会直接触发运行时错误并崩溃(编译期间)
  - 内存空间大小溢出
  - 申请的内存大于可以分配的内存
  - 传入的长度 小于 0 / len > cap
  
#### 内存申请
- runtime.makeslice 在最后调用的 runtime.mallocgc (用于申请内存的函数)
  - runtime.mallocgc 如果遇到了比较小的对象会直接初始化在 Go 语言调度器里面的 P 结构中，而大于 32KB 的对象会在堆上初始化
- 旧版本 ：数组指针、长度和容量会被合成一个 runtime.slice 结构
- 新版本1.16（cmd/compile: move slice construction to callers of makeslice 提交之后） ： 由调用方 runtime.makeslice 构建结构体 reflect.SliceHeader ，
  runtime.makeslice 仅会返回指向底层数组的指针，调用方会在编译期间构建切片结构体：
````go

func makeslice(et *_type, len, cap int) unsafe.Pointer {
  mem, overflow := math.MulUintptr(et.size, uintptr(cap))
    if overflow || mem > maxAlloc || len < 0 || len > cap {
      // NOTE: Produce a 'len out of range' error instead of a
      // 'cap out of range' error when someone does make([]T, bignumber).
      // 'cap out of range' is true too, but since the cap is only being
      // supplied implicitly, saying len is clearer.
      // See golang.org/issue/4085.
      mem, overflow := math.MulUintptr(et.size, uintptr(len))
      if overflow || mem > maxAlloc || len < 0 {
        panicmakeslicelen()
      }
      panicmakeslicecap()
    }
  
  return mallocgc(mem, et, true)
}

func typecheck1(n *Node, top int) (res *Node) {
	switch n.Op {
	...
	case OSLICEHEADER:
	switch 
		t := n.Type
		n.Left = typecheck(n.Left, ctxExpr)
		l := typecheck(n.List.First(), ctxExpr)
		c := typecheck(n.List.Second(), ctxExpr)
		l = defaultlit(l, types.Types[TINT])
		c = defaultlit(c, types.Types[TINT])

		n.List.SetFirst(l)
		n.List.SetSecond(c)
	...
	}
}
````
- OSLICEHEADER 会创建 reflect.SliceHeader，包含数组指针、切片长度和容量 . 它是切片在运行时的表示
```go
type SliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
```
- reflect.SliceHeader 的引入能够减少切片初始化时的少量开销
## 3.元素访问
- 编译器将 len 、 cap看成两种不同的操作， 即 OLEN 和 OCAP
- cmd/compile/internal/gc.state.expr  （ ssa.go - 2054 ）函数会在 SSA 生成阶段阶段将它们分别转换成 OpSliceLen 和 OpSliceCap
```go
func (s *state) expr(n *Node) *ssa.Value {
	switch n.Op {
	case OLEN, OCAP:
		switch {
		case n.Left.Type.IsSlice():
			op := ssa.OpSliceLen
			if n.Op == OCAP {
				op = ssa.OpSliceCap
			}
			return s.newValue1(op, types.Types[TINT], s.expr(n.Left))
		...
		}
	...
	}
}
```
- 访问切片中的字段可能会触发 “decompose builtin” 阶段的优化，len(slice) 或者 cap(slice) 在一些情况下会直接替换成切片的长度或者容量，不需要在运行时获取
```go
(SlicePtr (SliceMake ptr _ _ )) -> ptr
(SliceLen (SliceMake _ len _)) -> len
(SliceCap (SliceMake _ _ cap)) -> cap
```
- 访问切片中元素使用的 OINDEX 操作 在中间代码生成期间 转换成 对地址的直接访问：
```go
func (s *state) expr(n *Node) *ssa.Value {
	switch n.Op {
	case OINDEX:
		switch {
		case n.Left.Type.IsSlice():
			p := s.addr(n, false)
			return s.load(n.Left.Type.Elem(), p)
		...
		}
	...
	}
}
```
- 切片的操作基本都是在编译期间完成的，除了访问切片的长度、容量或者其中的元素之外，编译期间也会将包含 range 关键字的遍历转换成形式更简单的循环
## 4.追加和扩容
### append cap足够时
- 使用append进行追加， 中间代码生成阶段的方法( cmd/compile/internal/gc.state.append  | ssa.go- 2841) 会根据返回值是否覆盖原变量，进入两种流程
  - 1. append 返回的新切片不需要赋值回原有的变量, append(slice, 1, 2, 3)  
  - 2. append 后的切片会覆盖原切片 , slice = append(slice, 1, 2, 3) 语句
- 这两种最大的区别就是： 得到的新切片是否会赋值回原变量，方案2(覆盖原切片) 不用担心发生拷贝影响性能。
```go
// 1. append(slice, 1, 2, 3)
ptr, len, cap := slice
newlen := len + 3
if newlen > cap {
    ptr, len, cap = growslice(slice, newlen)
    newlen = len + 3
}
*(ptr+len) = 1
*(ptr+len+1) = 2
*(ptr+len+2) = 3
return makeslice(ptr, newlen, cap)
// 先获取切片结构体的指针， 大小， 容量； 如果追加后的切片的 len > cap,那么就会调用runtime.growslice 对切片进行扩容，将新元素一次加入切片

// 2. slice = append(slice, 1, 2, 3)
a := &slice
ptr, len, cap := slice
newlen := len + 3
if uint(newlen) > uint(cap) {
  newptr, len, newcap = growslice(slice, newlen)
  vardef(a)
  *a.cap = newcap
  *a.ptr = newptr
}
newlen = len + 3
*a.len = newlen
*(ptr+len) = 1
*(ptr+len+1) = 2
*(ptr+len+2) = 3
//  这种append 切片会覆盖原切片，这时 cmd/compile/internal/gc.state.append  会使用上述方法展开关键字 append 
```
#### append cap不够时
- 当切片的容量不足时， 调用 runtime.growslice 函数为切片扩容，扩容是为切片分配新的内存空间并拷贝原切片中元素的过程
1. 新切片的容量是如下确定的(第一部分)：
    - 如果期望容量大于当前容量的两倍就会使用期望容量；
    - 如果当前切片的长度小于 1024 就会将容量翻倍；
    - 如果当前切片的长度大于 1024 就会每次增加 25% 的容量，直到新容量大于期望容量
2. 第二部分考虑内存对齐
    - 根据切片中的元素大小对齐内存，当数组中元素所占的字节大小为 1、8 或者 2 的倍数时，运行时会使用如下所示的代码对齐内存
    - runtime.roundupsize 函数会将待申请的内存向上取整，使用 runtime.class_to_size （go/src/runtime/sizeclasses.go - 84 行）数组，
      使用该数组中的整数可以提高内存的分配效率并减少碎片
```go
// 1.
func growslice(et *_type, old slice, cap int) slice {
	...
	newcap := old.cap
	doublecap := newcap + newcap
	if cap > doublecap {
		newcap = cap
	} else {
		if old.len < 1024 {
			newcap = doublecap
		} else {
			for 0 < newcap && newcap < cap {
				newcap += newcap / 4
			}
			if newcap <= 0 {
				newcap = cap
			}
		}
	}
// 2.
// -------------- 下面第二部分	
	
  var overflow bool
  var lenmem, newlenmem, capmem uintptr
  switch {
  case et.size == 1:
    lenmem = uintptr(old.len)
    newlenmem = uintptr(cap)
    capmem = roundupsize(uintptr(newcap))
    overflow = uintptr(newcap) > maxAlloc
    newcap = int(capmem)
  case et.size == sys.PtrSize:
    lenmem = uintptr(old.len) * sys.PtrSize
    newlenmem = uintptr(cap) * sys.PtrSize
    capmem = roundupsize(uintptr(newcap) * sys.PtrSize)
    overflow = uintptr(newcap) > maxAlloc/sys.PtrSize
    newcap = int(capmem / sys.PtrSize)
  case isPowerOfTwo(et.size):
    var shift uintptr
    if sys.PtrSize == 8 {
      // Mask shift for better code generation.
      shift = uintptr(sys.Ctz64(uint64(et.size))) & 63
    } else {
      shift = uintptr(sys.Ctz32(uint32(et.size))) & 31
    }
    lenmem = uintptr(old.len) << shift
    newlenmem = uintptr(cap) << shift
    capmem = roundupsize(uintptr(newcap) << shift)
    overflow = uintptr(newcap) > (maxAlloc >> shift)
    newcap = int(capmem >> shift)
  default:
    lenmem = uintptr(old.len) * et.size
    newlenmem = uintptr(cap) * et.size
    capmem, overflow = math.MulUintptr(et.size, uintptr(newcap))
    capmem = roundupsize(capmem)
    newcap = int(capmem / et.size)
  }

}

```
3. 默认情况下，我们会将目标容量和元素大小相乘得到占用的内存。如果计算新容量时发生了内存溢出或者请求内存超过上限，就会直接崩溃退出程序
  - 切片中元素不是指针类型，那么会调用 runtime.memclrNoHeapPointers 将超出切片当前长度的位置清空并在最后使用 runtime.memmove 将原数组内存中的内容拷贝到新申请的内存中
```go

// 3. 
var overflow bool
var newlenmem, capmem uintptr
switch {
  ...
  default:
  lenmem = uintptr(old.len) * et.size
  newlenmem = uintptr(cap) * et.size
  capmem, _ = math.MulUintptr(et.size, uintptr(newcap))
  capmem = roundupsize(capmem)
  newcap = int(capmem / et.size)
}
...
var p unsafe.Pointer
  if et.kind&kindNoPointers != 0 {
    p = mallocgc(capmem, nil, false)
    memclrNoHeapPointers(add(p, newlenmem), capmem-newlenmem)
  } else {
    p = mallocgc(capmem, et, true)
    if writeBarrier.enabled {
      bulkBarrierPreWriteSrcOnly(uintptr(p), uintptr(old.array), lenmem)
    }
  }
memmove(p, old.array, lenmem)
return slice{p, old.len, newcap}

```
4. runtime.growslice 函数最终会返回一个新的切片，其中包含了新的数组指针、大小和容量，这个返回的三元组最终会覆盖原切片
## 5.拷贝切片
- 操作方式： copy(a,b)  --- copy(dst []Type, src []Type) 将b的值copy给a
- 编译期间的 cmd/compile/internal/gc.copyany 也会分两种情况进行处理拷贝操作
1.当前 copy 不是在运行时调用的，即编译期调用，则会被转换成：（runtime.memmove 会负责拷贝内存）
```go
n := len(a)
if n > len(b) {
    n = len(b)
}
if a.ptr != b.ptr {
    memmove(a.ptr, b.ptr, n*sizeof(elem(a))) 
}
```
2. 在运行时发生copy,编译器使用 runtime.slicecopy 替换运行期间调用的 copy
```go
func slicecopy(to, fm slice, width uintptr) int {
	if fm.len == 0 || to.len == 0 {
		return 0
	}
	n := fm.len
	if to.len < n {
		n = to.len
	}
	if width == 0 {
		return n
	}
	...

	size := uintptr(n) * width
	if size == 1 {
		*(*byte)(to.array) = *(*byte)(fm.array)
	} else {
		memmove(to.array, fm.array, size)
	}
	return n
}

```
## 6.总结