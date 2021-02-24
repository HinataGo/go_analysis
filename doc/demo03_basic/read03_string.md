# 字符串(只读的切片类型)
- 字符串的组成实际上由 它的底层 只读 字节数组组成， string 
- 只读的实现是 编译器将其标记为  只读数据 SRODATA 
- 只读只意味着字符串会分配到只读的内存空间,go只是不支持直接修改 ，但是可以通过转换为[]byte 修改后在转换
    - 先将这段内存拷贝到堆或者栈上；
    - 将变量的类型转换成 []byte 后并修改字节数据；
    - 将修改后的字节数组转换回 string；
```shell
// cd  string 
GOOS=linux GOARCH=amd64 go tool compile -S str.go
...
go.cuinfo.packagename. SDWARFINFO dupok size=0
        0x0000 73 74 72 69 6e 67                                string
go.string."world" SRODATA dupok size=5
        0x0000 77 6f 72 6c 64                                   world
...
```

- ps： go中两种特别的数据结构
```go
// aliases
	Byte = Uint8
	Rune = Int32
```
## 1. 数据结构
-  字符串底层由 reflect.StringHeader (go/src/reflect/value.go - 1983) 表示， 比切片仅仅少了一个cap
```go
// string
type StringHeader struct {
	Data uintptr
	Len  int
}
// slice 
type SliceHeader struct {
Data uintptr
Len  int
Cap  int
}
```
- 因为字符串的只读， 所以不直接追加元素改变其本身内存空间， 而是通过copy实现

## 2. 解析过程
- 解析器，在 词法分析阶段(解析字符串)： 对源文件中的字符进行切片和分组， 将原有无意义 字符流转换程Token序列
- 声明字符串由两种方式 ：
    - 双引号 ： 复杂多多
    - 反引号 : 推荐反引号，可直接使用 ",不需要\ 转义， 同时可以不受制单行限制，复杂的接送推荐使用
```go
str1 := "hello"
str2 := `wo
rld`
```
- 解析字符串使用的扫描器 cmd/compile/internal/syntax.scanner（scanner.go - 30） 会将输入的字符串转换成 Token 流
    - 1. 双引号字符串(标准字符串)： cmd/compile/internal/syntax.scanner.stdString （scanner.go - 669）
  - 标准字符串使用双引号表示开头和结尾；
  - 标准字符串需要使用反斜杠 \ 来转义双引号；
  - 标准字符串不能出现如下所示的隐式换行 \n；
      - 2. 反引号字符串(原始字符串)： cmd/compile/internal/syntax.scanner.rawString （scanner.go - 701），将非反引号的所有字符划分到当前字符串范围中 ，所以它支持复杂多行字符串
- 无论是标准字符串还是原始字符串都会被标记成 StringLit 并传递到语法分析阶段。在语法分析阶段，与字符串相关的表达式都会由 cmd/compile/internal/gc.noder.basicLit (noder.go - 1399)方法处理
    - 去除换行符 并对 Token 进行 Unquote ( go/src/strconv/quote.go - 368) (去掉字符串两边的引号等无关干扰)
```go
func (p *noder) basicLit(lit *syntax.BasicLit) Val {
	switch s := lit.Value; lit.Kind {
	case syntax.StringLit:
		if len(s) > 0 && s[0] == '`' {
			s = strings.Replace(s, "\r", "", -1)
		}
		u, _ := strconv.Unquote(s)
		return Val{U: u}
	}
}
```

## 3. 拼接
- 使用 + 符号时 编译器会将该符号对应的 OADD 节点转换成 OADDSTR 类型的节点，随后在 cmd/compile/internal/gc.walkexpr (walk.go - 411 )中调用 cmd/compile/internal/gc.addstr (walk.go - 2640)函数生成用于拼接字符串的代码：
```go
func walkexpr(n *Node, init *Nodes) *Node {
	switch n.Op {
	...
	case OADDSTR:
		n = addstr(n, init)
	}
}

```
- 编译期间 由 cmd/compile/internal/gc.addstr 选择合适的字符串拼接函数
    - 如果小于或者等于 5 个，那么会调用 concatstring{2,3,4,5} 等一系列函数；
    - 如果超过 5 个，那么会选择 runtime.concatstrings 传入一个数组切片；
```go
func addstr(n *Node, init *Nodes) *Node {
	c := n.List.Len()

	buf := nodnil()
	args := []*Node{buf}
	for _, n2 := range n.List.Slice() {
		args = append(args, conv(n2, types.Types[TSTRING]))
	}

	var fn string
	if c <= 5 {
		fn = fmt.Sprintf("concatstring%d", c)
	} else {
		fn = "concatstrings"

		t := types.NewSlice(types.Types[TSTRING])
		slice := nod(OCOMPLIT, nil, typenod(t))
		slice.List.Set(args[1:])
		args = []*Node{buf, slice}
	}

	cat := syslook(fn)
	r := nod(OCALL, cat, nil)
	r.List.Set(args)
	...

	return r
}
```
- 最终调用 runtime.concatstrings，先对遍历传入的切片，再过滤空字符串 并计算 拼接后字符串的长度
```go
func concatstrings(buf *tmpBuf, a []string) string {
	idx := 0
	l := 0
	count := 0
	for i, x := range a {
		n := len(x)
		if n == 0 {
			continue
		}
		l += n
		count++
		idx = i
	}
	if count == 0 {
		return ""
	}
	if count == 1 && (buf != nil || !stringDataOnStack(a[idx])) {
		return a[idx]
	}
	s, b := rawstringtmp(buf, l)
	for _, x := range a {
		copy(b, x)
		b = b[len(x):]
	}
	return s
}

```
- 非空字符串的数量为 1 并且当前的字符串不在栈上，就可以直接返回该字符串，不需要做出额外操作
- 运行时会调用 copy 将输入的多个字符串拷贝到目标字符串所在的内存空间。新的字符串是一片新的内存空间，与原来的字符串也没有任何关联，一旦需要拼接的字符串非常大，拷贝带来的性能损失是无法忽略的。
## 4. 类型转换
- string经常与[]byte进行转换 ，同时开销不小，使用 runtime.slicebytetostring ( go/src/runtime/string.go - 80)函数 
- 无论从哪种类型转换到另一种都需要拷贝数据，而内存拷贝的性能损耗会随着字符串和 []byte 长度的增长而增长
## 5. 总结