# interface
- 接口是计算机系统中多个组件共享的边界，不同的组件能够在边界上交换信息
- （上下游接口解耦）接口的本质是引入一个新的中间层，调用方可以通过接口与具体实现分离，解除上下游的耦合，上层的模块不再需要依赖下层的具体模块，只需要依赖一个约定好的接口
- 面向接口的编程方式， 在框架或者 基于POSIX （可移植操作系统接口 Portable Operating System Interface）设计的OS上很广泛应用
## 1 隐式接口
- 对比java ，java 必须显式的实现接口才可以使用。
- GO可以不用，GO开箱即用， GO定义接口使用 interface 关键字，接口中只能定义方法签名, 在 Go 中：实现接口的所有方法就隐式地实现了接口；
```go
type error interface {
	Error() string
}
```
- 代码interface 的 main.go，使用上述 RPCError 结构体时并不关心它实现了哪些接口，Go 语言只会在传递参数、返回参数以及变量赋值时才会对某个类型是否实现接口进行检查
- 这段代码在执行时，编译期间 及新宁三次代码检查:
    - 将 *RPCError 类型的变量赋值给 error 类型的变量 rpcErr；
    - 将 *RPCError 类型的变量 rpcErr 传递给签名中参数类型为 error 的 AsErr 函数；
    - 将 *RPCError 类型的变量从函数签名的返回值类型为 error 的 NewRPCError 函数中返回；
- 编译器仅在需要时才检查类型，类型实现接口时只需要实现接口中的全部方法，不需要像 Java 等编程语言中一样显式声明
## 2 类型
- (这两种都是使用interface 声明)Go 语言使用 runtime.iface 表示第一种接口，使用 runtime.eface 表示第二种不包含任何方法的接口 interface{}
- interface{} 不是任何类型，不同于c的 void*， 将类型转化为 interface{} 在运行期间也会发生变化，获取变量类型是会得地interface{}
- 参数接受interface{} 值时，Print 函数时会对参数 v 进行类型转换，将原来的 Test 类型转换成 interface{} 类型
```go
func main() {
	type Test struct{}
	v := Test{}
	Print(v)
}

func Print(v interface{}) {
	println(v)
}
```
## 3 指针 & 接口
- 接口在定义一方法时没有对实现的接收者做限制,所以常常有两种方法， 一种方法加接受者加指针 ，一种不加指针 ，因为结构体类型和指针类型是不同的
```go
// 对 Cat 结构体来说，它在实现接口时可以选择接受者的类型，即结构体或者结构体指针，
// 在初始化时也可以初始化成结构体或者指针。
// 下面的代码总结了如何使用结构体、结构体指针实现接口，以及如何使用结构体、结构体指针初始化变量。
type Cat struct {}
type Duck interface { ... }

func (c  Cat) Quack {}  // 使用结构体实现接口
func (c *Cat) Quack {}  // 使用结构体指针实现接口

var d Duck = Cat{}      // 使用结构体初始化变量
var d Duck = &Cat{}     // 使用结构体指针初始化变量

// 代码 interface有写

```
- 实现接口的类型和初始化返回的类型两个维度共组成了四种情况，然而这四种情况不是都能通过编译器的检查：
- 只记住（一种不能过的） ，接受者为指针，但是使用 结构体初始化指针吗，而非使用结构体指针：

| | 结构体实现接口 |结构体指针实现接口 |
|---|---|---|
|结构体初始化变量|通过|不通过|
|结构体指针初始化变量|通过|通过|
- &Cat{} 来说，这意味着拷贝一个新的 &Cat{} 指针，这个指针与原来的指针指向一个相同并且唯一的结构体，所以编译器可以隐式的对变量解引用（dereference）获取指针指向的结构体；
- Cat{} 来说，这意味着 Quack 方法会接受一个全新的 Cat{}，因为方法的参数是 *Cat，编译器不会无中生有创建一个新的指针；即使编译器可以创建新指针，这个指针指向的也不是最初调用该方法的结构体；


## 4 nil non-nil （Go 语言的接口类型不是任意类型）
- 这是由隐式转换地带来的一个问题
	- 将上述变量与 nil 比较会返回 true；
	- 将上述变量传入 NilOrNot 方法并与 nil 比较会返回 false；
- 发生原因 ： GO中在调用 参数为interface 的 函数时，会发生隐式转换，除了传入参数外，变量的赋值也会触发隐式类型转换，*TestStruct 类型会转换成 interface{} 类型，转换后的变量不仅包含转换前的变量，还包含变量的类型信息 TestStruct。所以转换后判断会有false
```go
type TestStruct struct{}

func NilOrNot(v interface{}) bool {
return v == nil
}

func main() {
var s *TestStruct
fmt.Println(s == nil)      // #=> true
fmt.Println(NilOrNot(s))   // #=> false
}
```
## 5 数据结构
- 使用 runtime.iface 结构体表示包含方法的接口
- 使用 runtime.eface 结构体表示不包含任何方法的 interface{} 类型
```go
type iface struct { // 16 字节
tab  *itab
data unsafe.Pointer
}

type eface struct { // 16 字节
_type *_type
data  unsafe.Pointer
}
```

### 类型结构体
- runtime._type 是 Go 语言类型的运行时表示。下面是运行时包中的结构体，其中包含了很多类型的元信息，例如：类型的大小、哈希、对齐以及种类等
```go
type _type struct {
size       uintptr
ptrdata    uintptr
hash       uint32
tflag      tflag
align      uint8
fieldAlign uint8
kind       uint8
equal      func(unsafe.Pointer, unsafe.Pointer) bool
gcdata     *byte
str        nameOff
ptrToThis  typeOff
}
// size 字段存储了类型占用的内存空间，为内存空间的分配提供信息；
// hash 字段能够帮助我们快速确定类型是否相等；
// equal 字段用于判断当前类型的多个对象是否相等，该字段是为了减少 Go 语言二进制包大小从 typeAlg 结构体中迁移过来的
```
### itab结构体
```go
type itab struct { // 32 字节
inter *interfacetype
_type *_type
hash  uint32
_     [4]byte
fun   [1]uintptr
}
```
- hash 是对 _type.hash 的拷贝，当我们想将 interface 类型转换成具体类型时，可以使用该字段快速判断目标类型和具体类型 runtime._type 是否一致；
- fun 是一个动态大小的数组，它是一个用于动态派发的虚函数表，存储了一组函数指针。虽然该变量被声明成大小固定的数组，但是在使用时会通过原始指针获取其中的数据，所以 fun 数组中保存的元素数量是不确定的
## 6 类型转换
### 指针类型
### 结构体类型
## 7 类型断言 （将接口转换为具体类型）
- xx.(type)
### 空接口
### 非空接口
## 8 动态派发
- 动态派发（Dynamic dispatch）是在运行期间选择具体多态操作（方法或者函数）执行的过程, Go的接口用于这一特性， 调用接口类型的方法时，如果编译器不能确定接口类型，GO语言会在运行期决定调用该方法的哪个实现
	- 第一次以 Duck 接口类型的身份调用，调用时需要经过运行时的动态派发；
	- 第二次以 *Cat 具体类型的身份调用，编译期就会确定调用的函数：
``` go
func main() {
var c Duck = &Cat{Name: "draven"}
c.Quack()
c.(*Cat).Quack()
}
```
- 使用-N禁止编译器优化，避免优化导致理解影响。
## 9总结