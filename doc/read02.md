## 编译原理学习
### 1.抽象语法树(AST) 
- 用树状的方式表示编程语言的语法结构
- eg: 2 * 3 + 7
- 波兰表示法
```md
                +
              /    \
            *       7
           /  \
          2    3
```
- 抽象语法树抹去了源代码中不重要的一些字符 - 空格、分号或者括号
###  2.静态单赋值
- 如果中间代码具有静态单赋值的特性，那么每个变量就只会被赋值一次
```go
x := 1
x := 2
y := x
```
- 中间代码（通过减少需要执行的指令优化这段代码）
    - 首先我们知道最终的y结果是 2 ，也就是和第一个 x 无关，因为第二个x会重新定义 x 并初始化值，
    编译器就会进行一个标 实现静态单赋值，随后优化，也就是不对 x:=1进行赋值（它被省略），
```go
x_1 := 1
x_2 := 2
y_1 := x_2
```
- SSA 的主要作用是对代码进行优化，所以它是编译器后端的一部分

### 3.指令集
- 本地开发环境编译和运行正常的代码，在生产环境却无法正常工作，本质就是不同机器使用不同的指令集
- Ubuntu可以输入 指令查看硬件信息
```shell
uname -m
```
- 指令集主要两大类
    - 复杂指令集(CISC)：通过增加指令的类型减少需要执行的指令数；
    - 精简指令集(RISC)：使用更少的指令类型完成目标的计算任务；
    
### 4.编译原理
- Go 语言编译器的源代码在 src/cmd/compile 目录，目录下的文件共同组成了 Go 语言的编译器
- 编译器分为前后端
    - 前端：词法分析、语法分析、类型检查和中间代码
    - 后端：目标代码的生成和优化，将中间代码翻译成目标机器能够运行的二进制机器码
- 过程：lexical -> syntax -> semantic -> IR generation -> code optimization -> machine code generation
- go的分为四步骤：
  - 词法与语法分析 
  - 类型检查
  - AST 转换、通用 SSA 生成 
  - 最后的机器代码生成
  
#### 4.1词法与语法分析
- 词法解析器 lexer
  - 编译过程都是从解析 代码的源码文件开始的，此法词法分析作用就是解析源代码文件，将文件中的字符串序列转换成token 序列（为后续处理和解析做准备）
  - 语法分析的输入 数据  是 才发分析器输出的 token序列，语法分析器会按照顺序解析token 序列，
    这个过程会将此法分析生成的token 按照编程语言定义好的 文法(Grammar）自下而上或者自上而下的规约
  - 每一个GO文件最终会被归纳成一个 SourceFile 结构
  ```shell
  SourceFile = PackageClause ";" { ImportDecl ";" } { TopLevelDecl ";" } .
  ```
  - 词法分析会返回一个不包含空格、换行等字符的 Token 序列
  ```json
  package,json,import,(, io, ), …
  ```
- 语法分析器（LALR(1) 的文法，）
  - 作用： Token -> 抽象语法树（AST）
  - 语法分析会把 Token 序列转换成有意义的结构体(语法树)
  ```json
  "json.go": SourceFile {
      PackageName: "json",
      ImportDecl: []Import{
          "io",
      },
      TopLevelDecl: ...
  }
  ```
  - 每一个 AST 都对应着一个单独的 Go 语言文件，这个抽象语法树中包括当前文件属于的包名、定义的常量、结构体和函数等

#### 4.2类型检查
- 类型检查两大过程（节点的类型进行验证，展开和改写一些内建的函数 ）
- 当拿到一组文件的AST后，Go会对AST种定义和使用的类型做检查，检查顺序
  -  常量、类型和函数名及类型；
  -  变量的赋值和初始化；
  -  函数和闭包的主体；
  -  哈希键值对的类型；
  -  导入函数体；
  -  外部的声明；
- （这一过程所有的类型错误和不匹配，都会被查到）对整个AST进行遍历，每个节点上都会对当前子树进行验证，保证节点类型不存再类型错误
- 改写展开和改写内建函数比如 make 根据子树结构被替换成： runtime.makeslice  runtime.makechan runtime.makemap 等等

#### 4.3中间代码生成
- 作用 AST(已检查) -> 中间代码
- 经历了前面源码文件 -> AST -> 类型检查，之后认为当前文件中，不存在语法错误 类型错误问题了。
- GO编译器就会输入AST转换成中间代码 ，
  这一过程通过  cmd/compile/internal/gc.compileFunctions  编译整个Go全部函数。
  这些函数会在一个编译队列中，等待几个Go程的消费，并发执行的goroutine 会将所有函数函数对应的抽象语法树转换成中间代码
- 中间代码具有SSA特性，因此会对 无用变量和片段并对代码进行优化

####4.4 机器码生成
- 指定文件生成机器码命令
```shell
cd demo01
GOARCH=wasm GOOS=js go build -o lib.wasm main.go
# lib.wasm 文件就是 WebAssembly 机器码
# 这个则是生辰amd64机器码
GOARCH=amd64 go build -o lib.amd64 main.go
```

- src/cmd/compile/internal 包含生成机器码相关代码，支持很多amd64、arm、arm64、mips、mips64、ppc64、s390x、x86 和 wasm


### 编译器入口
- Go 语言的编译器入口在 src/cmd/compile/internal/gc/main.go
- 600 多行的 cmd/compile/internal/gc.Main 就是 Go 语言编译器的主程序该函数会先获取命令行传入的参数并更新编译选项和配置，
  随后会调用 cmd/compile/internal/gc.parseFiles 对输入的文件进行词法与语法分析得到对应的抽象语法树
```go
func Main(archInit func(*Arch)) {
	...

	lines := parseFiles(flag.Args())
```
- 编译完整过程
```shell
    检查常量、类型和函数的类型；
    处理变量的赋值；
    对函数的主体进行类型检查；
    决定如何捕获变量；
    检查内联函数的类型；
    进行逃逸分析；
    将闭包的主体转换成引用的捕获变量；
    编译顶层函数；
    检查外部依赖的声明；
```