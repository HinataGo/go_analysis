# 哈希表
## 1 设计原理
### 1.1 哈希函数
### 1.2 哈希冲突
## 2 数据结构
### 2.1 go 中哈希结构 由 runtime.hmap ( go/src/runtime/map.go  - 115行) 决定 --  （go 的哈希表，是多个数据结构组合）
- count 表示当前哈希表中的元素数量；
- B 表示当前哈希表持有的 buckets 数量，但是因为哈希表中桶的数量都 2 的倍数，所以该字段会存储对数，也就是 len(buckets) == 2^B；
- hash0 是哈希的种子，它能为哈希函数的结果引入随机性，这个值在创建哈希表时确定，并在调用哈希函数时作为参数传入；
- oldbuckets 是哈希在扩容时用于保存之前 buckets 的字段，它的大小是当前 buckets 的一半；
```go
type hmap struct {
count     int
flags     uint8
B         uint8
noverflow uint16
hash0     uint32

buckets    unsafe.Pointer
oldbuckets unsafe.Pointer
nevacuate  uintptr

extra *mapextra
}

type mapextra struct {
overflow    *[]*bmap
oldoverflow *[]*bmap
nextOverflow *bmap
}
```
- (这两种桶在内存中是连续存储的)
  - 正常桶(buckets -> bmap): 哈希表 runtime.hmap 的桶是 runtime.bmap， 每一个 runtime.bmap 都能存储 8 个键值对，
  - 溢出桶(extra -> Overflow ): 当哈希表中存储的数据过多，单个桶已经装满时就会使用 extra.nextOverflow 中桶存储溢出的数据。由c语言设计，并且由于其鞥减少扩容的频率也一直在用。
#### 2.2.1 runtime.bmap
- 桶的结构体源码中 runtime.bmap 中 只定义一个 tophash 字段，包含 key哈希的最高8位，这是为 了 访问k-v时 比较哈希的高8位，减少访问次数，提升性能
```go
type bmap struct {
	tophash [bucketCnt]uint8
}
```
- 在运行期中， 由于go不支持泛型，并且哈希表中可能存储不同类型的k-v ，所以对于kv 占据的空间大小值，只能在编译期 推导。
- runtime.bmap 中的其他字段在运行时也都是通过计算内存地址的方式访问的，所以它的定义中就不包含这些字段 ，因此编译期间  cmd/compile/internal/gc.bmap 函数重建它的结构
- 因此运行经过编译后的运行期 ，runtime.bmap 中不只一个 tophash 字段
```go
type bmap struct {
    topbits  [8]uint8
    keys     [8]keytype
    values   [8]valuetype
    pad      uintptr
    overflow uintptr
}
```
- 当哈希表存储的数据逐渐增多，会扩容哈希表 / 使用额外的桶存储溢出的数据。不会让单个桶超过8个。但溢出桶知识临时解决方法，过多也会导致哈希扩容。

## 3 初始化
- 初始化方式 两种 字面量& 运行时
### 3.1 字面量 (key: value )【字面量初始化哈希也只是语言提供的辅助工具，最后调用的都是 runtime.makemap】
```go
hash := map[string]int{
"1": 2,
"3": 4,
"5": 6,
}
```
- 字面量初始化通过  cmd/compile/internal/gc.maplit 函数来进行
```go
func maplit(n *Node, m *Node, init *Nodes) {
	a := nod(OMAKE, nil, nil)
	a.Esc = n.Esc
	a.List.Set2(typenod(n.Type), nodintconst(int64(n.List.Len())))
	litas(m, a, init)

	entries := n.List.Slice()
	if len(entries) > 25 {
		...
		return
	}

	// Build list of var[c] = expr.
	// Use temporaries so that mapassign1 can have addressable key, elem.
	...
}
```
1. 当哈希表中的元素数量少于或者等于 25 个时，编译器会将字面量初始化的结构体转换成直接的kv赋值方式，一次性加入到hash表中 (集合类型的初始化在 Go 语言中有着相同的处理逻辑)
```go
hash := make(map[string]int, 3)
hash["1"] = 2
hash["3"] = 4
hash["5"] = 6
```
2. 哈希元素超过25时， 编译器会创建两个数组分别存储键和值，这些键值对会通过如下所示的 for 循环加入哈希
```go
hash := make(map[string]int, 26)
vstatk := []string{"1", "2", "3", ... ， "26"}
vstatv := []int{1, 2, 3, ... , 26}
for i := 0; i < len(vstak); i++ {
    hash[vstatk[i]] = vstatv[i]
}
```
- 切片 vstatk  vstatv会按照切片的凡是再展开
### 3.2 运行时
- 1.特殊优化当创建的哈希被分配到栈上并且其容量小于 BUCKETSIZE = 8 时，Go 语言在编译阶段会使用如下方式快速初始化哈希，这也是编译器对小容量的哈希做的优化：
```go
var h *hmap
var hv hmap
var bv bmap
h := &hv
b := &bv
h.buckets = b
h.hash0 = fashtrand0()
```
- 2.除1的情况，无其他使用make创建hash是，GO语言编译器都会  在类型检查期间转换成  runtime.makemap
```go
func makemap(t *maptype, hint int, h *hmap) *hmap {
	mem, overflow := math.MulUintptr(uintptr(hint), t.bucket.size)
	if overflow || mem > maxAlloc {
		hint = 0
	}

	if h == nil {
		h = new(hmap)
	}
	h.hash0 = fastrand()

	B := uint8(0)
	for overLoadFactor(hint, B) {
		B++
	}
	h.B = B

	if h.B != 0 {
		var nextOverflow *bmap
		h.buckets, nextOverflow = makeBucketArray(t, h.B, nil)
		if nextOverflow != nil {
			h.extra = new(mapextra)
			h.extra.nextOverflow = nextOverflow
		}
	}
	return h
}
```
1. 计算hash占用内存是否溢出 / 超过可分配最大值
2. 调用 runtime.fastrand 获取随机hash种子
3. 根据传入的 hint 计算出需要的桶数量
4. 使用runtime.makeBucketArray (map.go - 344)创建用于保存桶的数组
- runtime.makeBucketArray 会根据传入的 B 计算出的需要创建的桶数量并在内存中分配一片连续的空间用于存储数据：
  - 当桶的数量小于 24 时，由于数据较少、使用溢出桶的可能性较低，会省略创建的过程以减少额外开销； 
  - 当桶的数量多于 24 时，会额外创建 2B−4 个溢出桶；
```go
func makeBucketArray(t *maptype, b uint8, dirtyalloc unsafe.Pointer) (buckets unsafe.Pointer, nextOverflow *bmap) {
	base := bucketShift(b)
	nbuckets := base
	if b >= 4 {
		nbuckets += bucketShift(b - 4)
		sz := t.bucket.size * nbuckets
		up := roundupsize(sz)
		if up != sz {
			nbuckets = up / t.bucket.size
		}
	}

	buckets = newarray(t.bucket, int(nbuckets))
	if base != nbuckets {
		nextOverflow = (*bmap)(add(buckets, base*uintptr(t.bucketsize)))
		last := (*bmap)(add(buckets, (nbuckets-1)*uintptr(t.bucketsize)))
		last.setoverflow(t, (*bmap)(buckets))
	}
	return buckets, nextOverflow
}

```
- 正常桶和溢出桶在内存中的存储空间是连续的，只是被 runtime.hmap 中的不同字段引用，当溢出桶数量较多时会通过 runtime.newobject 创建新的溢出桶。

## 4 操作

- 哈希表的访问一般都是通过下标或者遍历进行的
```go
// 1. 需要知道哈希的键并且一次只能获取单个键对应的值
_ = hash[key]

// 2. 遍历哈希中的全部键值对，访问数据时也不需要预先知道哈希的键
for k, v := range hash {
    // k, v
}

```
- 基础操作
```go
var hash map[T]T
hash = make(map[T]T, size)
hash[key] = value
hash[key] = newValue
delete(hash, key)
// 一般还会查询key做这种操作
if v , OK := map[key]; OK {
	...
}
```
### 4.1 访问 （推荐使用两个返回值的 即 ， v,ok方式 ，ok是 bool值，用于检查key是否存在）
- 类型检查期间，hash[key] 以及类似的操作都会被转换成哈希的 OINDEXMAP 操作
- 中间代码生成阶段会在 cmd/compile/internal/gc.walkexpr 函数中将这些 OINDEXMAP 操作转换成如下的代码：
```go
v     := hash[key] // => v     := *mapaccess1(maptype, hash, &key)
v, ok := hash[key] // => v, ok := mapaccess2(maptype, hash, &key)
```
- 一个参数时，会使用 runtime.mapaccess1，该函数仅会返回一个指向目标值的指针；在扩容期间获取kv， 当哈希表中oldbuckets 存在时，会先定位到旧桶，并在该桶没有分流时从中回去kv
- 两个参数时，会使用 runtime.mapaccess2，除了返回目标值之外，它还会返回一个用于表示当前键对应的值是否存在的 bool 值(在 runtime.mapaccess1 的基础上多返回了一个标识键值对是否存在的 bool 值)
```go
// mapaccess1
func mapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	alg := t.key.alg
	hash := alg.hash(key, uintptr(h.hash0))
	m := bucketMask(h.B)
	b := (*bmap)(add(h.buckets, (hash&m)*uintptr(t.bucketsize)))
	top := tophash(hash)
bucketloop:
	for ; b != nil; b = b.overflow(t) {
		for i := uintptr(0); i < bucketCnt; i++ {
			if b.tophash[i] != top {
				if b.tophash[i] == emptyRest {
					break bucketloop
				}
				continue
			}
			k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
			if alg.equal(key, k) {
				v := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.valuesize))
				return v
			}
		}
	}
	return unsafe.Pointer(&zeroVal[0])
}

// mapaccess2
func mapaccess2(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, bool) {
  ...
  bucketloop:
  for ; b != nil; b = b.overflow(t) {
    for i := uintptr(0); i < bucketCnt; i++ {
      if b.tophash[i] != top {
        if b.tophash[i] == emptyRest {
            break bucketloop
        }
        continue
      }
      k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
        if alg.equal(key, k) {
            v := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.valuesize))
            return v, true
        }
    }
  }
  return unsafe.Pointer(&zeroVal[0]), false
}
```
- runtime.mapaccess1 会先通过哈希表设置的哈希函数、种子获取当前键对应的哈希，再通过 runtime.bucketMask 和 runtime.add 拿到该键值对所在的桶序号和哈希高位的 8 位数字
- 在 bucketloop 循环中，哈希会依次遍历正常桶和溢出桶中的数据，它会先比较哈希的高 8 位和桶中存储的 tophash，后比较传入的和桶中的值以加速数据的读写。用于选择桶序号的是哈希的最低几位，而用于加速访问的是哈希的高 8 位，这种设计能够减少同一个桶中有大量相等 tophash 的概率影响性能
#### tips 
- 哈希表扩容并不是原子过程，在扩容的过程中保证哈希的访问是一个重要 的问题，因此 普通的map不是并发安全的
### 4.2 写入
- 当进行hash写入操作时 即，hash[key] = xxx --> 编译期会转换成 runtime.mapassign 函数调用， 类似 mapaccess1
  - 先根据key拿到哈希和桶
  - 遍历比较桶中存储的 tophash 和key 的hash， 找到了相同结果就会返回目标位置的地址
  - inserti 表示 目标元素在桶中的索引，insertk 和 val 分贝别表示 k、v 的地址，获得目标地址后 会通过算术计算寻址获得键值对 kv
  - 标签bucketloop 的 for 循环会依次遍历正常桶和溢出桶中存储的数据，整个过程会分别判断 tophash 是否相等、key 是否相等，遍历结束后会从循环中跳出
```go
func mapassign(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	alg := t.key.alg
	hash := alg.hash(key, uintptr(h.hash0))

	h.flags ^= hashWriting

again:
	bucket := hash & bucketMask(h.B)
	b := (*bmap)(unsafe.Pointer(uintptr(h.buckets) + bucket*uintptr(t.bucketsize)))
	top := tophash(hash)
// ----------------------------------------------------

  var inserti *uint8
  var insertk unsafe.Pointer
  var elem unsafe.Pointer
bucketloop:
  for {
    for i := uintptr(0); i < bucketCnt; i++ {
      if b.tophash[i] != top {
        if isEmpty(b.tophash[i]) && inserti == nil {
          inserti = &b.tophash[i]
          insertk = add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
          elem = add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
        }
        if b.tophash[i] == emptyRest {
          break bucketloop
        }
      continue
      }
      k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
        if t.indirectkey() {
           k = *((*unsafe.Pointer)(k))
        }
        if !t.key.equal(key, k) {
          continue
        }
    // already have a mapping for key. Update it.
        if t.needkeyupdate() {
          typedmemmove(t.key, k, key)
        }
        elem = add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
        goto done
    }
    ovf := b.overflow(t)
    if ovf == nil {
      break
    }
    b = ovf
  }
```
#### 溢出情况 （当前已经满的情况下）
- 哈希会调用 runtime.hmap.newoverflow  (go/src/runtime/map.go -245)创建新桶或者使用 runtime.hmap 预先在 noverflow 中创建好的桶来保存数据，新创建的桶不仅会被追加到已有桶的末尾，还会增加哈希表的 noverflow 计数器
```go
if inserti == nil {
		newb := h.newoverflow(t, b)
		inserti = &newb.tophash[0]
		insertk = add(unsafe.Pointer(newb), dataOffset)
		val = add(insertk, bucketCnt*uintptr(t.keysize))
	}

	typedmemmove(t.key, insertk, key)
	*inserti = top
	h.count++

done:
	return val
}
```
1. 如果当前kv 哈希表中不存在，hash 会为新建的 kv 规划存储内存地址，通过  runtime.typedmemmove  (go/src/runtime/mbarrier.go - 156)将键移动到对应的内存空间中并返回键对应值的地址 val
2. 如果存在， 直接返回目标区域的内存地址，哈希并不会在 runtime.mapassign 这个运行时函数中将值拷贝到桶中，该函数只会返回内存地址，真正的赋值操作是在编译期间插入的
```go
// 24(SP) 是该函数返回的值地址
// LEAQ 指令将字符串的地址存储到寄存器 AX 中
// MOVQ 指令将字符串 "88" 存储到了目标地址上完成了这次哈希的写入
00018 (+5) CALL runtime.mapassign_fast64(SB)
00020 (5) MOVQ 24(SP), DI               ;; DI = &value
00026 (5) LEAQ go.string."88"(SB), AX   ;; AX = &"88"
00027 (5) MOVQ AX, (DI)                 ;; *DI = AX

```
### 4.3 扩容
- 哈希写入过程时其实省略了扩容操作，随着哈希表中元素的逐渐增加，哈希的性能会逐渐恶化，所以我们需要更多的桶和更大的内存保证哈希的读写性能
- 1.扩容情况 两种扩容机制（根据发生的原因不同，操作不同）（等量扩容 和 翻倍扩容）
  - 装载因子已经超过 6.5 (翻倍扩容 runtime.growWork )
  - 哈希使用了太多溢出桶  (等量扩容)
- 2.哈希扩容非原子操作 ，会先判断是否处于扩容状态，避免二次扩容混乱
```go
func mapassign(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	...
	if !h.growing() && (overLoadFactor(h.count+1, h.B) || tooManyOverflowBuckets(h.noverflow, h.B)) {
		hashGrow(t, h)
		goto again
	}
	...
}

```
- 扩容详解:
  - (溢出桶太多->扩容， 进行等量扩容) sameSizeGrow （ go/src/runtime/map.go - 1026） ，持续向哈希中插入数据并将它们全部删除:
    - 如果哈希表中的数据没有超过阈值，就会不断积累溢出桶造成缓慢的内存泄露 （[runtime: limit the number of map overflow buckets](https://github.com/golang/go/commit/9980b70cb460f27907a003674ab1b9bea24a847c) ）
    - sameSizeGrow 通过复用已有的哈希扩容机制 解决上述问题， 一旦哈希中出现了过多的溢出桶，它会创建新桶保存数据，垃圾回收会清理老的溢出桶并释放内存
    - 单独再创建一个新桶，初始化一个runtime.evacDst， 旧桶与新桶之间是一对一的关系， 该函数中只是创建了新的桶，并没有对数据进行拷贝和转移
  - (范翻倍扩容) ，runtime.evacuate 将原桶的数据 一分为二 ，所以创建两个  保存分配上下文的 runtime.evacDst 结构体，并且两个结构体指向一个新桶
- 扩容时 通过runtime.makeBucketArray 创建一组 新桶 & 预创建的溢出桶，然后将 原有桶数组设置 到oldbuckets ，将新的空桶设置到 buckets
- 溢出桶相同， 原先的溢出桶设置到oldoverflow ， 预创建溢出桶放在 nextoverflow 中
- ps ：  溢出桶都是在 mapextra中定义 ，正常桶 （buckets oldbuckets）都在 hmap 中定义，一起的数据结构还包括 count， flag B ，nevacuate， extra 
```go
func hashGrow(t *maptype, h *hmap) {
	bigger := uint8(1)
	if !overLoadFactor(h.count+1, h.B) {
		bigger = 0
		h.flags |= sameSizeGrow
	}
	oldbuckets := h.buckets
	newbuckets, nextOverflow := makeBucketArray(t, h.B+bigger, nil)

	h.B += bigger
	h.flags = flags
	h.oldbuckets = oldbuckets
	h.buckets = newbuckets
	h.nevacuate = 0
	h.noverflow = 0

	h.extra.oldoverflow = h.extra.overflow
	h.extra.overflow = nil
	h.extra.nextOverflow = nextOverflow
}
```
#### Tips
- 翻倍扩容 & 等量扩容区别
- 翻倍在于需要创建新的 两个桶，然后拷贝转移数据，分流到两个桶，这是要创建两个保存分配上下文 的 runtime.evacDst 结构体，并是它们指向一个新桶。
- 等量吗， 在于新创建一个桶，与原桶对应起来，随后，不需要数据复制转移， 后续添加元素，直接往对应的新桶操作
- 哈希表的数据迁移的过程在是 runtime.evacuate 中完成的，它会对传入桶中的元素进行再分配
```go
func evacuate(t *maptype, h *hmap, oldbucket uintptr) {
	b := (*bmap)(add(h.oldbuckets, oldbucket*uintptr(t.bucketsize)))
	newbit := h.noldbuckets()
	if !evacuated(b) {
		var xy [2]evacDst
		x := &xy[0]
		x.b = (*bmap)(add(h.buckets, oldbucket*uintptr(t.bucketsize)))
		x.k = add(unsafe.Pointer(x.b), dataOffset)
		x.v = add(x.k, bucketCnt*uintptr(t.keysize))

		y := &xy[1]
		y.b = (*bmap)(add(h.buckets, (oldbucket+newbit)*uintptr(t.bucketsize)))
		y.k = add(unsafe.Pointer(y.b), dataOffset)
		y.v = add(y.k, bucketCnt*uintptr(t.keysize))
```

```go
// 翻倍扩容逻辑 
for ; b != nil; b = b.overflow(t) {
  k := add(unsafe.Pointer(b), dataOffset)
  v := add(k, bucketCnt*uintptr(t.keysize))
  for i := 0; i < bucketCnt; i, k, v = i+1, add(k, uintptr(t.keysize)), add(v, uintptr(t.valuesize)) {
    top := b.tophash[i]
    k2 := k
    var useY uint8
    hash := t.key.alg.hash(k2, uintptr(h.hash0))
      if hash&newbit != 0 {
        useY = 1
      }
    b.tophash[i] = evacuatedX + useY
    dst := &xy[useY]
    
    if dst.i == bucketCnt {
      dst.b = h.newoverflow(t, dst.b)
      dst.i = 0
      dst.k = add(unsafe.Pointer(dst.b), dataOffset)
      dst.v = add(dst.k, bucketCnt*uintptr(t.keysize))
    }
    dst.b.tophash[dst.i&(bucketCnt-1)] = top
    typedmemmove(t.key, dst.k, k)
    typedmemmove(t.elem, dst.v, v)
    dst.i++
    dst.k = add(dst.k, uintptr(t.keysize))
    dst.v = add(dst.v, uintptr(t.valuesize))
  }
}
```
- 只使用哈希函数是不能定位到具体某一个桶的，哈希函数只会返回很长的哈希， 一般都会使用取模或者位操作来获取桶的编号，如果四个桶那就是 0b11， 8个桶就是 0b111
  - 如果新的哈希表有 8 个桶，在大多数情况下，原来经过桶掩码 0b11 结果为 3 的数据会因为桶掩码增加了一位编程 0b111 而分流到新的 3 号和 7 号桶， 数据也都会被 runtime.typedmemmove 拷贝到目标桶中：
```go
// 具体操作为
hash code & 0bxx 
```
- runtime.evacuate 最后会调用 runtime.advanceEvacuationMark 增加哈希的 nevacuate 计数器并在所有的旧桶都被分流后清空哈希的 oldbuckets 和 oldoverflow：
```go
func advanceEvacuationMark(h *hmap, t *maptype, newbit uintptr) {
	h.nevacuate++
	stop := h.nevacuate + 1024
	if stop > newbit {
		stop = newbit
	}
	for h.nevacuate != stop && bucketEvacuated(t, h, h.nevacuate) {
		h.nevacuate++
	}
	if h.nevacuate == newbit { // newbit == # of oldbuckets
		h.oldbuckets = nil
		if h.extra != nil {
			h.extra.oldoverflow = nil
		}
		h.flags &^= sameSizeGrow
	}
}

```
- runtime.mapassign 在哈希表处于扩容时访问，每次向哈希表写入值，会触发，runtime.growWork 增量拷贝哈希表中内容
  - 删除类似 ，逻辑上都是 先计算当前值所在的桶，再拷贝桶中的元素
```go
func mapassign(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	...
again:
	bucket := hash & bucketMask(h.B)
	if h.growing() {
		growWork(t, h, bucket)
	}
	...
}
```
### 4.4 删除
- delete ，将一个kv 从hash 中删除，无论kv是否存储在，不返回任何结果
- 编译期 ， delete转换成 ODELETE 的节点， 并且被 cmd/compile/internal/gc.walkexpr 转换成 runtime.mapdelete 函数簇中的一个，包括 runtime.mapdelete、mapdelete_faststr、mapdelete_fast32 和 mapdelete_fast64
```go
func walkexpr(n *Node, init *Nodes) *Node {
	switch n.Op {
	case ODELETE:
		init.AppendNodes(&n.Ninit)
		map_ := n.List.First()
		key := n.List.Second()
		map_ = walkexpr(map_, init)
		key = walkexpr(key, init)

		t := map_.Type
		fast := mapfast(t)
		if fast == mapslow {
			key = nod(OADDR, key, nil)
		}
		n = mkcall1(mapfndel(mapdelete[fast], t), nil, init, typename(t), map_, key)
	}
}

```