# for & range 
## 1.示例
### 循环的range遍历
- 在遍历切片时追加的元素不会增加循环的执行次数，所以循环最终还是停了下来
```go
func main() {
	arr := []int{1, 2, 3}
	for _, v := range arr {
		arr = append(arr, v)
	}
	fmt.Println(arr)
}
// 1 2 3 1 2 3
```
### range指针
- 获取 range 返回变量的地址并保存到另一个数组或者哈希时
- 正确的做法应该是使用 &arr[i] 替代 &v
```go
func main() {
	arr := []int{1, 2, 3}
	newArr := []*int{}
	for _, v := range arr {
		newArr = append(newArr, &v)
	}
	for _, v := range newArr {
		fmt.Println(*v)
	}
}
// 输出 “3 3 3”
```
### 遍历清空数组
- 遍历清空非常浪费，但是 编译器会直接使用 runtime.memclrNoHeapPointers 清空切片中的数据 （数组、切片和哈希占用的内存空间都是连续的，所以最快的方法是直接清空这片内存中的内容）
```go
func main() {
	arr := []int{1, 2, 3}
	for i, _ := range arr {
		arr[i] = 0
	}
}

```

### 随机遍历
- Go 语言中使用 range 遍历哈希表时,每次结果都不同
- Go 语言在运行时为哈希表的遍历引入了不确定性，程序不要依赖于哈希表的稳定遍历
```go
func main() {
    hash := map[string]int{
        "1": 1,
        "2": 2,
        "3": 3,
    }
    for k, v := range hash {
        println(k, v)
    }
}
```
## 2.一般循环
#### 一个for循环在编译器看来是一个 OFOR类型节点
1. 初始化循环的 Ninit；
2. 循环的继续条件 Left；
3. 循环体结束时执行的 Right；
4. 循环体 NBody：
```go
for Ninit; Left; Right {
    NBody
}

```
####  一个常见的 for 循环代码会被 cmd/compile/internal/gc.state.stmt 转成相应的结构

## 3. 范围循环
- 循环同时使用 for 和 range 两个关键字，编译器会在编译期间将所有 for-range 循环变成经典循环。从编译器的视角来看，就是将 ORANGE 类型的节点转换成 OFOR 节点:
### array & slice
- cmd/compile/internal/gc.walkrange 函数将 对于数组 slice的遍历由三种转换方式    
- 分析遍历数组和切片清空元素的情况；
    - 分析使用 for range a {} 遍历数组和切片，不关心索引和数据的情况；
    - 分析使用 for i := range a {} 遍历数组和切片，只关心索引的情况；
    - 分析使用 for i, elem := range a {} 遍历数组和切片，关心索引和数据的情况；

```go
func walkrange(n *Node) *Node {
	switch t.Etype {
    case TARRAY, TSLICE:
        if arrayClear(n, v1, v2, a) {
            return n
        }
        ...
}
```
- cmd/compile/internal/gc.arrayClear, 优化 Go 语言遍历数组或者切片并删除全部元素的逻辑：
```go
// 原代码
for i := range a {
	a[i] = zero
}

// 优化后
if len(a) != 0 {
	hp = &a[0]
	hn = len(a)*sizeof(elem(a))
	memclrNoHeapPointers(hp, hn)
	i = len(a) - 1
}
```
#### 清空数据
- Go 语言会直接使用 runtime.memclrNoHeapPointers 或者 runtime.memclrHasPointers 清除目标数组内存空间中的全部数据，并在执行完成后更新遍历数组的索引
### hash
- 编译器会使用 runtime.mapiterinit 和 runtime.mapiternext 两个运行时函数重写原始的 for-range 循环
```go
for key, val := range hash {
	...
}
// 展开
ha := a
hit := hiter(n.Type)
th := hit.Type
mapiterinit(typename(t), ha, &hit)
for ; hit.key != nil; mapiternext(&hit) {
    key := *hit.key
    val := *hit.val
}
 // 随后 在 cmd/compile/internal/gc.walkrange 处理 TMAP 节点时, 编译器会根据 range 返回值的数量在循环体中插入需要的赋值语句
```
### string(底层 rune)
- 在遍历时会获取字符串中索引对应的字节并将字节转换成 rune。我们在遍历字符串时拿到的值都是 rune 类型的变量
```go
for i, r := range s {}
// 实际结构
ha := s
for hv1 := 0; hv1 < len(ha); {
    hv1t := hv1
    hv2 := rune(ha[hv1])
    if hv2 < utf8.RuneSelf {
        hv1++
    } else {
    	hv2, hv1 = decoderune(ha, hv1)
    }
    v1, v2 = hv1t, hv2
}
```
- 使用下标访问字符串中的元素时得到的就是字节，但是这段代码会将当前的字节转换成 rune 类型。如果当前的 rune 是 ASCII 的，那么只会占用一个字节长度，每次循环体运行之后只需要将索引加一，但是如果当前 rune 占用了多个字节就会使用 runtime.decoderune 函数解码
### channel
```go
var ch chan
for v := range ch {}
// 转换后
ha := a
hv1, hb := <-ha
for ; hb != false; hv1, hb = <-ha {
v1 := hv1
hv1 = nil
...
}
```
- 该循环会使用 <-ch 从管道中取出等待处理的值，这个操作会调用 runtime.chanrecv2 并阻塞当前的协程，当 runtime.chanrecv2 返回时会根据布尔值 hb 判断当前的值是否存在：
  - 如果不存在当前值，意味着当前的管道已经被关闭
  - 如果存在当前值，会为 v1 赋值并清除 hv1 变量中的数据，然后重新陷入阻塞等待新数据