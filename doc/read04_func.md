# 函数
## 1 调用过程
```go
func main() {
	op(10,10)
}
func op(a,b int ) int{
	return a + b
}

```
- 针对 function.go 文件 使用 : go tool compile -S -N -l function.go
```go

// 生成如下汇编
"".main STEXT size=71 args=0x0 locals=0x20
0x0000 00000 (function.go:3)    TEXT    "".main(SB), ABIInternal, $32-0
0x0000 00000 (function.go:3)    MOVQ    (TLS), CX
0x0009 00009 (function.go:3)    CMPQ    SP, 16(CX)
0x000d 00013 (function.go:3)    PCDATA  $0, $-2
0x000d 00013 (function.go:3)    JLS     64
0x000f 00015 (function.go:3)    PCDATA  $0, $-1
0x000f 00015 (function.go:3)    SUBQ    $32, SP
0x0013 00019 (function.go:3)    MOVQ    BP, 24(SP)
0x0018 00024 (function.go:3)    LEAQ    24(SP), BP
0x001d 00029 (function.go:3)    FUNCDATA        $0, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
0x001d 00029 (function.go:3)    FUNCDATA        $1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
0x001d 00029 (function.go:4)    MOVQ    $10, (SP)
0x0025 00037 (function.go:4)    MOVQ    $10, 8(SP)
0x002e 00046 (function.go:4)    PCDATA  $1, $0
0x002e 00046 (function.go:4)    CALL    "".op(SB)
0x0033 00051 (function.go:5)    MOVQ    24(SP), BP
0x0038 00056 (function.go:5)    ADDQ    $32, SP
0x003c 00060 (function.go:5)    RET
0x003d 00061 (function.go:5)    NOP
0x003d 00061 (function.go:3)    PCDATA  $1, $-1
0x003d 00061 (function.go:3)    PCDATA  $0, $-2
0x003d 00061 (function.go:3)    NOP
0x0040 00064 (function.go:3)    CALL    runtime.morestack_noctxt(SB)
0x0045 00069 (function.go:3)    PCDATA  $0, $-1
0x0045 00069 (function.go:3)    JMP     0
...
"".op STEXT nosplit size=25 args=0x18 locals=0x0
0x0000 00000 (function.go:6)    TEXT    "".op(SB), NOSPLIT|ABIInternal, $0-24
0x0000 00000 (function.go:6)    FUNCDATA        $0, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
0x0000 00000 (function.go:6)    FUNCDATA        $1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
0x0000 00000 (function.go:6)    MOVQ    $0, "".~r2+24(SP)
0x0009 00009 (function.go:7)    MOVQ    "".a+8(SP), AX
0x000e 00014 (function.go:7)    ADDQ    "".b+16(SP), AX
0x0013 00019 (function.go:7)    MOVQ    AX, "".~r2+24(SP)
0x0018 00024 (function.go:7)    RET

```
- 函数调用栈会由 bp sp 指针控制，函数数据的，这在操作系统里
- 函数从右到左， 压栈入参
- 当我们准备好函数的入参之后，会调用汇编指令 CALL "".op(SB)，这个指令首先会将 main 的返回地址存入栈中，然后改变当前的栈指针 SP 并执行 op 的汇编指令

#### tip
- Go语言的方式能够降低实现的复杂度并支持多返回值，但是牺牲了函数调用的性能； 
- 不需要考虑超过寄存器数量的参数应该如何传递；
- 不需要考虑不同架构上的寄存器差异；
- 函数入参和出参的内存空间需要在栈上进行分配；
- Go 语言使用栈作为参数和返回值传递的方法是综合考虑后的设计，选择这种设计意味着编译器会更加简单、更容易维护。

## 2 参数参数传递（传值 or 传址 ？）
### 传值传应引用对比
- 传值：函数调用时会对参数进行拷贝，被调用方和调用方两者持有不相关的两份数据； （ 重点 拷贝）
- 传引用：函数调用时会传递参数的指针，被调用方和调用方两者持有相同的数据，任意一方做出的修改都会影响另一方。 （重点 会更改原电数据）
- GO本身就是传值 ，不存在传址
### 2.1 array （传值）
```go
before calling - i=(1, 0xc0000b6010) arr=([10 20], 0xc0000b6020)
in my_funciton - i=(1, 0xc0000b6018) arr=([10 20], 0xc0000b6040)
after  calling - i=(1, 0xc0000b6010) arr=([10 20], 0xc0000b6020)

```
- 运行 array.go 查看到  main 和 myFunction 参数的 地址 不同 
- myFunction 调用前后 i 和 arr两个参数地址不变
- 尝试在myFunction 中更改数据值 （去掉注释）
```go
before calling - i=(1, 0xc00001e0c0) arr=([10 20], 0xc00001e0d0)
in my_funciton - i=(30, 0xc00001e0c8) arr=([10 40], 0xc00001e0f0)
after  calling - i=(1, 0xc00001e0c0) arr=([10 20], 0xc00001e0d0)
```
- 地址仍然不变，同时只影响自己函数内 myFunction ，对于main函数 无影响，所以传值 
- 调用函数时会对内容进行拷贝。需要注意的是如果当前数组的大小非常的大，这种传值的方式会对性能造成比较大的影响。
### 2.2 struct pointer 
- 运行struct
```go
before calling - a=({10}, 0xc0000b6010) b=(&{20}, 0xc0000b8018)
in my_function - a=({100}, 0xc0000b6028) b=(&{200}, 0xc0000b8028)
after calling  - a=({10}, 0xc0000b6010) b=(&{200}, 0xc0000b8018)
```
- 传递结构体时：会拷贝结构体中的全部内容；
- 传递结构体指针时：会拷贝结构体指针；(类似传值)
    - 修改结构体指针是改变了指针指向的结构体，b.i 可以被理解成 (*b).i
#### 分析
- 指针修改结构体中的成员变量，结构体在内存中是一片连续的空间，指向结构体的指针也是指向这个结构体的首地址。
- 可以使用go tool compile 编译查看汇编结果
- 当参数是指针时，也会使用 MOVQ "".ms+8(SP), AX 指令复制引用，然后将复制后的指针作为返回值传递回调用方。
## 3 总结
- GO中不存在所谓深拷贝 
- GO中传值 不传址， 区别
- GO中的区别在于传递的时候给的是地址还是值
- chan 给的地址，所以不必使用 * ， 其他数据结构传地址 用*