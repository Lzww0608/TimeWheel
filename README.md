# TimeWheel


## Standalone TimeWheel
## 时间轮单机版本说明

`time_wheel_standalone.go` 实现了一个单机的时间轮调度器，用于高效管理和执行定时任务。

### 核心结构

#### taskElement 结构体
表示定时任务元素：
```go
type taskElement struct {
    task  func()    // 待执行的任务函数
    pos   int       // 在时间轮中的位置
    cycle int       // 需要经过的轮数
    key   string    // 任务的唯一标识符
}
```

#### TimeWheel 结构体
时间轮实现：
```go
type TimeWheel struct {
    sync.Once
    interval       time.Duration     // 时间轮的基本时间间隔
    ticker         *time.Ticker      // 定时器
    stopc          chan struct{}     // 停止信号通道
    addTaskCh      chan *taskElement // 添加任务的通道
    removeTaskCh   chan string       // 删除任务的通道
    slots          []*list.List      // 时间轮的槽，每个槽是一个任务链表
    slotNum        int               // 时间轮的槽数量
    currentSlot    int               // 当前指向的槽位置
    taskMap        map[string]*taskElement // 任务映射表，用于快速查找任务
}
```

### 主要方法

#### 初始化与控制

- `NewTimeWheel(slotNum int, interval time.Duration) *TimeWheel`: 
  创建并初始化一个新的时间轮，参数分别是槽数量和基本时间间隔。

- `Stop()`: 
  停止时间轮，释放相关资源。

#### 任务管理

- `AddTask(key string, task func(), executeAt time.Time)`: 
  添加一个定时任务，指定执行时间。

- `RemoveTask(key string)`: 
  根据任务标识符移除任务。

#### 内部实现方法

- `run()`: 
  时间轮的主循环，处理各种通道事件。

- `tick()`: 
  时间轮的一次滴答操作，执行当前槽位的到期任务。

- `circularIncr()`: 
  循环增加当前槽位指针。

- `execute(list *list.List)`: 
  执行指定链表中到期的任务。

- `getPosAndCycle(executeAt time.Time) (pos int, cycle int)`: 
  计算任务应该放在哪个槽位和经过多少轮后执行。

- `addTask(task *taskElement)`: 
  内部方法，将任务添加到时间轮中。

- `removeTask(key string)`: 
  内部方法，从时间轮中移除任务。

### 工作原理

1. **时间轮结构**：
   时间轮由多个槽位组成，每个槽位是一个任务链表。时间轮以固定的间隔（interval）转动，每次转动指向下一个槽位。

2. **任务调度**：
   - 当添加任务时，根据任务的执行时间计算它应该放在哪个槽位以及需要经过多少轮。
   - 时间轮每转动一次，检查当前槽位的任务，如果任务的轮数（cycle）为0，则执行该任务。

3. **并发控制**：
   - 使用通道（channel）进行事件驱动和并发控制。
   - 所有对时间轮的操作（添加任务、删除任务、停止时间轮）都通过相应的通道进行。

4. **任务执行**：
   - 到期的任务会在独立的goroutine中执行，避免阻塞时间轮的主循环。
   - 执行过程中会捕获任务可能产生的panic，确保时间轮的稳定运行。

### 实现细节

1. **时间计算**：
   - 使用毫秒级别的时间精度计算任务位置和周期
   - 处理负值延迟的情况，保证不会出现负数循环

2. **任务存储**：
   - 使用双向链表（`container/list`）存储每个槽位的任务
   - 使用map实现O(1)时间复杂度的任务查找

3. **并发安全**：
   - 所有对时间轮的操作都通过channel进行，避免直接访问内部状态
   - 任务执行在独立的goroutine中，不会阻塞主循环

### 使用示例

#### 基本使用

```go
// 创建一个有60个槽、每秒转动一次的时间轮
tw := NewTimeWheel(60, time.Second)

// 添加一个30秒后执行的任务
executeAt := time.Now().Add(30 * time.Second)
tw.AddTask("task1", func() {
    fmt.Println("Task executed!")
}, executeAt)

// 移除任务
tw.RemoveTask("task1")

// 停止时间轮
defer tw.Stop()
```

#### 并发使用场景

```go
tw := NewTimeWheel(10, 100*time.Millisecond)

// 使用互斥锁保护共享数据
var mu sync.Mutex
results := make(map[string]bool)

// 添加多个任务
for i := 0; i < 100; i++ {
    taskID := fmt.Sprintf("task-%d", i)
    executeAt := time.Now().Add(200 * time.Millisecond)
    
    tw.AddTask(taskID, func() {
        mu.Lock()
        defer mu.Unlock()
        results[taskID] = true
    }, executeAt)
}

// 等待所有任务执行完毕
time.Sleep(500 * time.Millisecond)
```

### 常见问题及解决方案

1. **任务未执行问题**
   - **症状**：添加的任务没有被执行
   - **可能原因**：执行时间已过期、时间轮已停止、任务被错误移除
   - **解决方案**：检查执行时间是否合理，确保时间轮未停止

2. **并发安全问题**
   - **症状**：遇到"concurrent map writes"等错误
   - **解决方案**：使用互斥锁保护在任务中对共享资源的访问

3. **性能调优**
   - **槽数量选择**：槽数越多，时间精度越高，但内存占用也越大
   - **时间间隔选择**：间隔越小，调度越精准，但CPU开销越大
   - **建议**：根据实际需求平衡槽数量和间隔大小，一般推荐60-3600个槽

4. **资源泄漏问题**
   - **症状**：长时间运行后内存占用不断增加
   - **解决方案**：确保不再需要的时间轮调用`Stop()`方法释放资源

### 适用场景

时间轮特别适合于以下场景：
1. 大量定时任务的高效管理
2. 需要精确控制执行时间的场景
3. 定时任务的动态添加和移除
4. 对性能要求较高的定时调度系统




