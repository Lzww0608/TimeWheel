package timewheel

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewTimeWheel(t *testing.T) {
	tw := NewTimeWheel(60, 1*time.Second)
	if tw == nil {
		t.Fatal("NewTimeWheel should not return nil")
	}
	if tw.interval != 1*time.Second {
		t.Errorf("Expected interval to be 1s, got %v", tw.interval)
	}
	if tw.slotNum != 60 {
		t.Errorf("Expected slotNum to be 60, got %d", tw.slotNum)
	}
	if len(tw.slots) != 60 {
		t.Errorf("Expected 60 slots, got %d", len(tw.slots))
	}
}

func TestAddTask(t *testing.T) {
	tw := NewTimeWheel(10, 100*time.Millisecond)
	
	// 添加任务测试
	taskExecuted := false
	task := func() {
		taskExecuted = true
	}
	
	// 延迟200ms执行
	executeAt := time.Now().Add(200 * time.Millisecond)
	tw.AddTask("task1", task, executeAt)
	
	// 等待任务执行
	time.Sleep(500 * time.Millisecond)
	
	if !taskExecuted {
		t.Error("Task was not executed")
	}
}

func TestRemoveTask(t *testing.T) {
	tw := NewTimeWheel(10, 100*time.Millisecond)
	
	taskExecuted := false
	task := func() {
		taskExecuted = true
	}
	
	taskID := "task1"
	executeAt := time.Now().Add(300 * time.Millisecond)
	tw.AddTask(taskID, task, executeAt)
	tw.RemoveTask(taskID)
	
	// 等待足够长的时间确保任务不会执行
	time.Sleep(500 * time.Millisecond)
	
	if taskExecuted {
		t.Error("Task should have been removed but was executed")
	}
}

func TestTaskExecution(t *testing.T) {
	tw := NewTimeWheel(10, 100*time.Millisecond)
	
	executionCount := 0
	task := func() {
		executionCount++
	}
	
	// 添加一次性任务
	executeAt := time.Now().Add(150 * time.Millisecond)
	tw.AddTask("task1", task, executeAt)
	
	// 等待任务执行
	time.Sleep(300 * time.Millisecond)
	
	if executionCount != 1 {
		t.Errorf("Expected task to execute once, but executed %d times", executionCount)
	}
}

func TestStopTimeWheel(t *testing.T) {
	tw := NewTimeWheel(10, 100*time.Millisecond)
	
	taskExecuted := false
	task := func() {
		taskExecuted = true
	}
	
	executeAt := time.Now().Add(200 * time.Millisecond)
	tw.AddTask("task1", task, executeAt)
	
	// 立即停止时间轮
	tw.Stop()
	
	// 等待一段时间
	time.Sleep(300 * time.Millisecond)
	
	if taskExecuted {
		t.Error("Task should not be executed after timewheel is stopped")
	}
}

func TestConcurrentOperations(t *testing.T) {
	tw := NewTimeWheel(10, 100*time.Millisecond)
	
	const taskCount = 100
	executedTasks := make(map[int]bool)
	var mu sync.Mutex  // 添加互斥锁保护map
	
	// 添加多个任务
	for i := 0; i < taskCount; i++ {
		i := i // 捕获变量
		executeAt := time.Now().Add(200 * time.Millisecond)
		// 使用strconv.Itoa正确转换整数到字符串
		taskKey := "task-" + strconv.Itoa(i)
		tw.AddTask(taskKey, func() {
			mu.Lock()         // 加锁保护map写入
			executedTasks[i] = true
			mu.Unlock()       // 解锁
		}, executeAt)
	}
	
	// 等待所有任务执行
	time.Sleep(500 * time.Millisecond)
	
	// 检查是否所有任务都已执行
	mu.Lock()  // 读取map前加锁
	tasksExecuted := len(executedTasks)
	mu.Unlock()
	
	if tasksExecuted != taskCount {
		t.Errorf("Expected %d tasks to execute, but %d executed", taskCount, tasksExecuted)
	}
}
