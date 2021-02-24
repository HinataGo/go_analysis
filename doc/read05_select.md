# select
- select 是操作系统中的系统调用。 select、poll 和 epoll 等函数构建 I/O 多路复用模型提升程序的性能。
- Go 语言中的 select 也能够让 Goroutine 同时等待多个 Channel 可读或者可写，在多个文件或者 Channel状态改变之前，select 会一直阻塞当前线程或者 Goroutine
- select与 switch区别在于 case 必须是 channel且必须表达为收发操作
    - 这段代码会等待 c <- x 或者 <-quit 两个表达式中任意一个返回。无论哪一个表达式返回都会立刻执行 case 中的代码，当 select 中的两个 case 同时被触发时，会随机执行其中的一个。
```go
func fibonacci(c, quit chan int) {
	x, y := 0, 1
	for {
		select {
		case c <- x:
			x, y = y, x+y
		case <-quit:
			fmt.Println("quit")
			return
		}
	}
}

```
### 两种常见情况
- select 能在 Channel 上进行非阻塞的收发操作 (非阻塞收发)
- select 在遇到多个 Channel 同时响应时，会随机执行一种情况 (随机执行)
## 非阻塞收发
- 一般， select 语句会阻塞当前 Goroutine 并等待多个 Channel 中的一个达到可以收发的状态。
- 当有default case 时， select会执行如下
    - 存在对应的channel出发， 执行对应case
    - 不存在对应channel情况 ，执行default
- 很多场景下我们不希望 Channel 操作阻塞当前 Goroutine，只是想看看 Channel 的可读或者可写状态
```go
errCh := make(chan error, len(tasks))
wg := sync.WaitGroup{}
wg.Add(len(tasks))
for i := range tasks {
    go func() {
        defer wg.Done()
        if err := tasks[i].Run(); err != nil {
            errCh <- err
        }
    }()
}
wg.Wait()

select {
case err := <-errCh:
    return err
default:
    return nil
}
```
## 随机执行
```go
func main() {
	ch := make(chan int)
	go func() {
		for range time.Tick(1 * time.Second) {
			ch <- 0
		}
	}()

	for {
		select {
		case <-ch:
			println("case1")
		case <-ch:
			println("case2")
		}
	}
}
```
## 数据结构
- runtime.scase 表示
- 非默认的 case 中都与 Channel 的发送和接收有关，所以 runtime.scase 结构体中也包含一个 runtime.hchan 类型的字段存储 case 中使用的 Channel
```go
type scase struct {
	c    *hchan         // chan
	elem unsafe.Pointer // data element
}
```

## 实现原理
```go

```