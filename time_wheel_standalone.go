package timewheel

import (
	"container/list"
	"sync"
	"time"
	"log"
)

type taskElement struct {
	task  func() 
	pos   int 
	cycle int 
	key   string 
}


type TimeWheel struct {
	sync.Once
	interval 		time.Duration
	ticker 			*time.Ticker 
	stopc	    	chan struct{}
	addTaskCh   	chan *taskElement
	removeTaskCh 	chan string
	slots 			[]*list.List
	slotNum 		int
	currentSlot 	int
	taskMap 		map[string]*taskElement
}


func NewTimeWheel(slotNum int, interval time.Duration) *TimeWheel {
	if slotNum <= 0 {
		slotNum = 10
	}

	if interval <= 0 {
		interval = time.Second
	}

	t := TimeWheel {
		interval: 	interval, 
		ticker: 	time.NewTicker(interval), 
		stopc: 		make(chan struct{}), 
		addTaskCh: 	make(chan *taskElement), 
		removeTaskCh: make(chan string), 
		slots: 		make([]*list.List, slotNum), 
		slotNum: 	slotNum, 
		currentSlot: 0, 
		taskMap: make(map[string]*taskElement), 
	}

	for i := 0; i < slotNum; i++ {
		t.slots[i] = list.New()
	}

	go t.run()
	return &t
}


func (t *TimeWheel) Stop() {
	t.Do(func() {
		t.ticker.Stop()
		close(t.stopc)
	})
}


func (t *TimeWheel) AddTask(key string, task func(), executeAt time.Time) {
	pos, cycle := t.getPosAndCycle(executeAt)
	t.addTaskCh <- &taskElement{
		task: task, 
		pos: pos, 
		cycle: cycle, 
		key: key, 
	}
}

func (t *TimeWheel) RemoveTask(key string) {
	t.removeTaskCh <- key
}

func (t *TimeWheel) run() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("timewheel panic: %v", err)
		}
	}()

	for {
		select {
		case <-t.stopc:
			return 
		case <-t.ticker.C:
			t.tick()
		case task := <-t.addTaskCh:
			t.addTask(task)
		case removeKey := <-t.removeTaskCh:
			t.removeTask(removeKey)
		}
	}


}

func (t *TimeWheel) tick() {
	list := t.slots[t.currentSlot]
	defer t.circularIncr()
	
	t.execute(list)
}

func (t *TimeWheel) circularIncr() {
	t.currentSlot = (t.currentSlot + 1) % t.slotNum
}

func (t *TimeWheel) execute(list *list.List) {
	for e := list.Front(); e != nil; {
		task, _ := e.Value.(*taskElement)
		if task.cycle > 0 {
			task.cycle--
			e = e.Next()
			continue
		}

		go func(task *taskElement) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("timewheel task panic: %v", err)
				}
			}()
			task.task()
		}(task)

		next := e.Next()
		list.Remove(e)
		e = next
		delete(t.taskMap, task.key)
	}
}


func (t *TimeWheel) getPosAndCycle(executeAt time.Time) (pos int, cycle int) {
	delay := int(time.Until(executeAt).Milliseconds())
	// 确保delay不为负
	if delay < 0 {
		delay = 0
	}
	
	intervalMs := int(t.interval.Milliseconds())
	cycle = delay / (t.slotNum * intervalMs)
	pos = (t.currentSlot + (delay / intervalMs) % t.slotNum) % t.slotNum
	return
}

func (t *TimeWheel) addTask(task *taskElement) {
	list := t.slots[task.pos]
	if _, ok := t.taskMap[task.key]; ok {
		t.removeTask(task.key)
	}

	t.taskMap[task.key] = task
	list.PushBack(task)
}

func (t *TimeWheel) removeTask(key string) {
	if task, ok := t.taskMap[key]; ok {
		delete(t.taskMap, key)
		for e := t.slots[task.pos].Front(); e != nil; e = e.Next() {
			if taskEle, ok := e.Value.(*taskElement); ok && taskEle.key == key {
				t.slots[task.pos].Remove(e)
				break
			}
		}
	}
}