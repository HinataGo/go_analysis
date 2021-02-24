# 数组
- 数组的表示：两个维度--元素类型和数组最大能存储的元素个数
```go
[10]int
[10]interface{}
```
- Go数组初始化之后大小就无法改变
- GO元素类型相同，但大小不一样的GO数组 是完全不同的， 被视为不同类型 | GO同意类型数组，必须 1.类型 2. 大小都一样
```go
func NewArray(elem *Type, bound int64) *Type {
	if bound < 0 {
		Fatalf("NewArray: invalid bound %v", bound)
	}
	t := New(TARRAY)
	t.Extra = &Array{Elem: elem, Bound: bound}
	t.SetNotInHeap(elem.NotInHeap())
	return t
}
```

### 初始化
1. 显式的指定数组大小 [N] T
2. 使用 [...]T 声明数组，Go 语言会在编译期间通过源代码推导数组的大小
```go
arr1 := [3]int{1, 2, 3}
arr2 := [...]int{1, 2, 3}
```
### 上限推导
1. 使用初始化第一种方式 - [N]T    ，变量类型会在编译的 类型检查阶段提取出来，随后使用 cmd/compile/internal/types.NewArray (type.go - 473)
   创建包含数组大小的 cmd/compile/internal/types.Array (type.go - 333 ) 结构体
```go
// Array contains Type fields specific to array types.
type Array struct {
    Elem  *Type // element type
    Bound int64 // number of elements; <0 if unknown yet
}

// NewArray returns a new fixed-length array Type.
func NewArray(elem *Type, bound int64) *Type {
	if bound < 0 {
		Fatalf("NewArray: invalid bound %v", bound)
	}
	t := New(TARRAY)
	t.Extra = &Array{Elem: elem, Bound: bound}
	t.SetNotInHeap(elem.NotInHeap())
	return t
}

```
2. 使用第二种初始化方式 - [...]T  , 编译器会在的 cmd/compile/internal/gc.typecheckcomplit (typecheck.go - 2792行) 函数中对该数组的大小进行推导
```go
# gc.typecheckcomplit
func typecheckcomplit(n *Node) (res *Node) {
	...
	if n.Right.Op == OTARRAY && n.Right.Left != nil && n.Right.Left.Op == ODDD {
		n.Right.Right = typecheck(n.Right.Right, ctxType)
		if n.Right.Right.Type == nil {
			n.Type = nil
			return n
		}
		elemType := n.Right.Right.Type

		length := typecheckarraylit(elemType, -1, n.List.Slice(), "array literal")

		n.Op = OARRAYLIT
		n.Type = types.NewArray(elemType, length)
		n.Right = nil
		return n
	}
	...

	switch t.Etype {
	case TARRAY:
		typecheckarraylit(t.Elem(), t.NumElem(), n.List.Slice(), "array literal")
		n.Op = OARRAYLIT
		n.Right = nil
	}
}
```
-  cmd/compile/internal/gc.typecheckcomplit 会调用 cmd/compile/internal/gc.typecheckarraylit 通过遍历元素的方式来计算数组中元素的数量
- 因此 ，[...]T{1, 2, 3} 和 [3]T{1, 2, 3} 在运行时是完全等价的，
  [...]T 这种初始化方式也只是 Go 语言为我们提供的一种语法糖，当我们不想计算数组中的元素个数时可以通过这种方法减少一些工作量。
  
### 语句转换
- 一个有字面量组成的数组，根据元素数量不同，编译器函数cmd/compile/internal/gc.anylit ， 会做两种不同优化
```go 
func anylit(n *Node, var_ *Node, init *Nodes) {
	t := n.Type
	switch n.Op {
	case OSTRUCTLIT, OARRAYLIT:
		if n.List.Len() > 4 {
			...
		}

		fixedlit(inInitFunction, initKindLocalCode, n, var_, init)
	...
	}
}
```
1. 当元素数量小于或者等于 4 个时，会直接将数组中的元素放置在栈上；
- 1.1 cmd/compile/internal/gc.fixedlit 会在函数编译之前将 [3]{1, 2, 3} 转换成更加原始的语句
```go
func fixedlit(ctxt initContext, kind initKind, n *Node, var_ *Node, init *Nodes) {
	var splitnode func(*Node) (a *Node, value *Node)
	...

	for _, r := range n.List.Slice() {
		a, value := splitnode(r)
		a = nod(OAS, a, value)
		a = typecheck(a, ctxStmt)
		switch kind {
		case initKindStatic:
			genAsStatic(a)
		case initKindLocalCode:
			a = orderStmtInPlace(a, map[string][]*Node{})
			a = walkstmt(a)
			init.Append(a)
		}
	}
}
```
- 1.2 当数组中元素的个数小于或者等于四个并且 cmd/compile/internal/gc.fixedlit 函数接收的 kind 是 initKindLocalCode 时，上述代码会将原有的初始化语句 [3]int{1, 2, 3} 拆分成一个声明变量的表达式和几个赋值表达式，这些表达式会完成对数组的初始化：
```go
var arr [3]int
arr[0] = 1
arr[1] = 2
arr[2] = 3
```
2. 当元素数量大于 4 个时，会将数组中的元素放置到静态区并在运行时取出；
- 当前数组的元素大于四个，cmd/compile/internal/gc.anylit 会先获取一个唯一的 staticname，然后调用 cmd/compile/internal/gc.fixedlit 函数在静态存储区初始化数组中的元素并将临时变量赋值给数组：
```go
func anylit(n *Node, var_ *Node, init *Nodes) {
	t := n.Type
	switch n.Op {
	case OSTRUCTLIT, OARRAYLIT:
		if n.List.Len() > 4 {
			vstat := staticname(t)
			vstat.Name.SetReadonly(true)

			fixedlit(inNonInitFunction, initKindStatic, n, vstat, init)

			a := nod(OAS, var_, vstat)
			a = typecheck(a, ctxStmt)
			a = walkexpr(a, init)
			init.Append(a)
			break
		}

		...
	}
}
// 同理  [5]int{1, 2, 3, 4, 5}，会被初始化为类似一下伪代码
var arr [5]int
statictmp_0[0] = 1
statictmp_0[1] = 2
statictmp_0[2] = 3
statictmp_0[3] = 4
statictmp_0[4] = 5
arr = statictmp_0

```
#### 小结
- 不考虑逃逸分析的情况下，如果数组中元素的个数小于或者等于 4 个，那么所有的变量会直接在栈上初始化，如果数组元素大于 4 个，变量就会在静态存储区初始化然后拷贝到栈上，这些转换后的代码才会继续进入中间代码生成和机器码生成两个阶段，最后生成可以执行的二进制文件。

### 访问和赋值
#### 越界检查，编译器期 和 运行期
- 无论在栈上还是静态存储区中，数组都是一段连续的内存。
- 如果我们不知道数组中元素的数量，访问时可能发生越界；而如果不知道数组中元素类型的大小，就没有办法知道应该一次取出多少字节的数据，无论丢失了那个信息，我们都无法知道这片连续的内存空间到底存储了什么数据：
- 数组操作越界是很严重的，go 中在编译期间有静态类型检查判断检查数组越界 ，cmd/compile/internal/gc.typecheck1 会验证访问数组的索引
```go
func typecheck1(n *Node, top int) (res *Node) {
	switch n.Op {
	case OINDEX:
		ok |= ctxExpr
		l := n.Left  // array
		r := n.Right // index
		switch n.Left.Type.Etype {
		case TSTRING, TARRAY, TSLICE:
			...
			if n.Right.Type != nil && !n.Right.Type.IsInteger() {
				yyerror("non-integer array index %v", n.Right)
				break
			}
			if !n.Bounded() && Isconst(n.Right, CTINT) {
				x := n.Right.Int64()
				if x < 0 {
					yyerror("invalid array index %v (index must be non-negative)", n.Right)
				} else if n.Left.Type.IsArray() && x >= n.Left.Type.NumElem() {
					yyerror("invalid array index %v (out of bounds for %d-element array)", n.Right, n.Left.Type.NumElem())
				}
			}
		}
	...
	}
}
```
- 访问数组的索引是非整数时，报错 “non-integer array index %v”；
- 访问数组的索引是负数时，报错 “invalid array index %v (index must be non-negative)"；
- 访问数组的索引越界时，报错 “invalid array index %v (out of bounds for %d-element array)"；

- 数组和字符串可以直接访问，使用整数或者常数访问发生错误会被直接发现，但是 变量会躲避这一编译期 发现错误，只会在运行期间发现
```go
arr[4]: invalid array index 4 (out of bounds for 3-element array)
arr[i]: panic: runtime error: index out of range [4] with length 3
```
- Go运行时由  runtime.panicIndex 和 runtime.goPanicIndex 出发运行时错误 panic (数组、切片和字符串 的越界操作)
```go
TEXT runtime·panicIndex(SB),NOSPLIT,$0-8
MOVL	AX, x+0(FP)
MOVL	CX, y+4(FP)
JMP	runtime·goPanicIndex(SB)

func goPanicIndex(x int, y int) {
panicCheck1(getcallerpc(), "index out of range")
panic(boundsError{x: int64(x), signed: true, y: y, code: boundsIndex})
}

```
- 数组的访问操作 OINDEX 成功通过编译器的检查后，会被转换成几个 SSA 指令,尝试编译array.go 
```shell
GOSSAFUNC=outOfRange go build array.go
```
- ssa.html 文件中可以查看到， elem := arr[i] 对应的中间代码，
  数组的访问操作生成了判断数组上限的指令 IsInBounds 
  以及当条件不满足时触发程序崩溃的 PanicBounds 指令
```plan9_x86
b1:
    ...
    v22 (6) = LocalAddr <*[3]int> {arr} v2 v20
    v23 (6) = IsInBounds <bool> v21 v11
If v23 → b2 b3 (likely) (6)

b2: ← b1-
    v26 (6) = PtrIndex <*int> v22 v21
    v27 (6) = Copy <mem> v20
    v28 (6) = Load <int> v26 v27 (elem[int])
    ...
Ret v30 (+7)

b3: ← b1-
    v24 (6) = Copy <mem> v20
    v25 (6) = PanicBounds <mem> [0] v21 v11 v24
Exit v25 (6)
```
- 编译器会将 PanicBounds 指令转换成 runtime.panicIndex 函数
    - 当下标没有越界时，编译器会先获取数组的内存地址和访问的下标、利用 PtrIndex 计算出目标元素的地址，最后使用 Load 操作将指针中的元素加载到内存中
    - 无法判断越界时，加入PanicBounds 指令交给运行时进行判断，字面量整数访问数组下标时会生成非常简单的中间代码
    - 将arr[i] 改成 arr[2] 时：
```plan9_x86
b1:
    ...
    v21 (5) = LocalAddr <*[3]int> {arr} v2 v20
    v22 (5) = PtrIndex <*int> v21 v14
    v23 (5) = Load <int> v22 v20 (elem[int])
    ...
```
#### 赋值和更新操作
- 数组的寻址，赋值 都是在编译阶段完成的，没有运行时的参与
- 赋值是先确定目标数组的地址，再通过PtrIndex 获取目标元素的地址， 最后使用Store 指令将数据存入地址中。
- 赋值和更新操作会 在生成SSA期间 计算数组当前的元素的内存地址，然后修改当前内存地址的内容， 这个赋值语句会被转换成如下SSA代码
```plan9_x86
b1:
    ...
    v21 (5) = LocalAddr <*[3]int> {arr} v2 v19
    v22 (5) = PtrIndex <*int> v21 v13
    v23 (5) = Store <mem> {int} v22 v20 v19
    ...
```
#### 小结
- GO对数组访问的检查，它不仅会在编译期间提前发现一些简单的越界错误并插入用于检测数组上限的函数调用，还会在运行期间通过插入的函数保证不会发生越界
### tips
- 数组元素数量大于4，在静态储存区分配内存时，该内存能够被回收吗？（不在堆上的内存如何回收）
    - 静态存储区的内存是不可变的，只有堆上和栈上的内存会回收
- 数组元素数量小于等于4，在栈上分配内存，那么如果函数返回数组地址，内存不会逃逸吗？
    - 会逃逸
```go
package main

import "fmt"

//go:noinline
func newArray() *[4]int {
    a := [4]int{1, 2, 3, 4}
    return &a
}
func main() {
    a := newArray()
    fmt.Println(a)
}

$ go build -gcflags='-m' main.go
# command-line-arguments
./main.go:13:13: inlining call to fmt.Println
./main.go:7:2: moved to heap: a                 // 逃逸
./main.go:13:13: []interface {} literal does not escape
<autogenerated>:1: .this does not escape

// 可以改为使用两个 -m 更加详细查看逃逸分析
go build -gcflags="-m -m" hello.go
# command-line-arguments
./hello.go:6:6: cannot inline newArray: marked go:noinline
./hello.go:11:6: cannot inline main: function too complex: cost 139 exceeds budget 80
./hello.go:13:13: inlining call to fmt.Println func(...interface {}) (int, error) { var fmt..autotmp_3 int; fmt..autotmp_3 = <N>; var fmt..autotmp_4 error; fmt..autotmp_4 = <N>; fmt..autotmp_3, fmt..autotmp_4 = fmt.Fprintln(io.Writer(os.Stdout), fmt.a...); return fmt..autotmp_3, fmt..autotmp_4 }
./hello.go:7:2: a escapes to heap:
./hello.go:7:2:   flow: ~r0 = &a:
./hello.go:7:2:     from &a (address-of) at ./hello.go:8:9
./hello.go:7:2:     from return &a (return) at ./hello.go:8:2
./hello.go:7:2: moved to heap: a                                // 逃逸
./hello.go:13:13: []interface {} literal does not escape
<autogenerated>:1: .this does not escape
```
[逃逸原因](https://golang.org/doc/faq#stack_or_heap)
1. 编译器发现在后续代码中存在对此局部变量的引用，因此将变量 a 移动到了 heap 中。
2. 在可能的情况下，编译器会在当前函数的栈中为局部变量分配内存。 一旦当编译器发现后续代码中存在对局部变量的引用，就会将局部变量从栈移动到堆，以此来避免 C/C++ 中常出现的所谓“悬空指针”的现象。
3. 当局部变量非常大的时候，编译器也会考虑在堆上创建局部变量。
4. 所以说，到底是在栈还是在堆上分配内存，并没有一个一成不变的规则，编译器会根据具体的情况做出最优选择。