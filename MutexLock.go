package program

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

//
// 1、Hacked Lock/TryLock 模式
//
//其实，对于标准库的sync.Mutex要增加这个功能很简单，下面的方式就是通过hack的方式为Mutex实现了TryLock的功能。

const mutexLocked = 1 << iota
type Mutex struct {
	mu sync.Mutex
}
func (m *Mutex) Lock() {
	m.mu.Lock()
}
func (m *Mutex) Unlock() {
	m.mu.Unlock()
}
func (m *Mutex) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&m.mu)), 0, mutexLocked)
}
func (m *Mutex) IsLocked() bool {
	return atomic.LoadInt32((*int32)(unsafe.Pointer(&m.mu))) == mutexLocked
}

//2、TryLock By Channel
//
//既然标准库中不准备在Mutex上增加这个方法，而是推荐使用channel来实现，那么就让我们看看如何使用 channel来实现。

type MutexChan struct {
	ch chan struct{}
}
func NewMutexChan() *MutexChan {
	mu := &MutexChan{make(chan struct{}, 1)}
	mu.ch <- struct{}{}
	return mu
}
func (m *MutexChan) Lock() {
	<-m.ch
}
func (m *MutexChan) Unlock() {
	select {
	case m.ch <- struct{}{}:
	default:
		panic("unlock of unlocked mutex")
	}
}
func (m *MutexChan) TryLock() bool {
	select {
	case <-m.ch:
		return true
	default:
	}
	return false
}
func (m *MutexChan) IsLocked() bool {
	return len(m.ch) > 0
}


//3、TryLock with Timeout
//
//有时候，我们在获取一把锁的时候，由于有竞争的关系，在锁被别的goroutine拥有的时候，当前goroutine没有办法立即获得锁，只能阻塞等待。标准库并没有提供等待超时的功能，我们尝试实现它。

type MutexTimeOut struct {
	ch chan struct{}
}
func NewMutexTimeOut() *Mutex {
	mu := &MutexTimeOut{make(chan struct{}, 1)}
	mu.ch <- struct{}{}
	return mu
}
func (m *MutexTimeOut) Lock() {
	<-m.ch
}
func (m *MutexTimeOut) Unlock() {
	select {
	case m.ch <- struct{}{}:
	default:
		panic("unlock of unlocked mutex")
	}
}
func (m *MutexTimeOut) TryLock(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	select {
	case <-m.ch:
		timer.Stop()
		return true
	case <-time.After(timeout):
	}
	return false
}
func (m *MutexTimeOut) IsLocked() bool {
	return len(m.ch) > 0
}