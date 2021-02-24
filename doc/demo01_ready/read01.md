# 源码解析01 demo01
[下载go1.15](https://github.com/golang/go/releases/tag/go1.15.8)
go的type类型定义可以看 /src/go/types/type.go
编译的 可以看  /src/cmd/compile/internal/types/type.go
## 调试go语言 
```shell
# 进入源码包路径下
cloc ./src
# 输出
    6072 text files.
    5937 unique files.                                          
    1188 files ignored.
github.com/AlDanial/cloc v 1.82  T=27.07 s (180.6 files/s, 71936.2 lines/s)
-----------------------------------------------------------------------------------
Language                         files          blank        comment           code
-----------------------------------------------------------------------------------
Go                                4260         144133         224502        1425687
Assembly                           486          12795          19158         106694
C                                   64            718            562           4587
JSON                                12              0              0           1712
Perl                                11            177            175           1106
Bourne Shell                         7            132            630           1042
Markdown                             6            230              0            714
Bourne Again Shell                  13            110            228            507
Python                               1            132            104            370
DOS Batch                            5             57              1            258
C/C++ Header                         9             56            158            142
Windows Resource File                4             23              0            139
RobotFramework                       1              0              0            106
C++                                  1              8              9             17
Objective C                          1              2              3             11
make                                 4              3              7              7
awk                                  1              1              6              7
MATLAB                               1              1              0              4
CSS                                  1              0              0              1
HTML                                 1              0              0              1
-----------------------------------------------------------------------------------
SUM:                              4889         158578         245543        1543112
-----------------------------------------------------------------------------------


```

## 源码编译
```shell
# 编译 Go 语言的二进制、工具链以及标准库和命令并将源代码和编译好的二进制文件移动到对应的位置上
./src/make.bash
# 编译好的二进制会存储在 $GOPATH/src/github.com/golang/go/bin 目录中 需要使用绝对路径来访问并使用它
$GOPATH/src/github.com/golang/go/bin/go run main.go
```

## 中间代码
- Go 语言的应用程序在运行之前需要先编译成二进制，在编译的过程中会经过中间代码生成阶段
- Go 语言编译器的中间代码具有静态单赋值特性（Static Single Assignment、SSA）
- 使用下面的命令将 Go 语言的源代码编译成汇编语言
```shell
go build -gcflags -S main.go
# 生成的汇编文件 main
```
#### 更详细的编译结果
```shell
# 执行命令
$ GOSSAFUNC=main go build main.go
## 下面两步骤自动进行
# runtime
dumped SSA to /usr/local/Cellar/go/1.14.2_1/libexec/src/runtime/ssa.html
# command-line-arguments
dumped SSA to ./ssa.html
```
#### # 最终生成文件
- ssa.html
- 通过浏览器可以查看
- 并且该文件可以交互