package program

import (
	"reflect"
	"sync"
)

//当你等待多个信号的时候，如果收到任意一个信号， 就执行业务逻辑，忽略其它的还未收到的信号。
//
//举个例子， 我们往提供相同服务的n个节点发送请求，只要任意一个服务节点返回结果，我们就可以执行下面的业务逻辑，其它n-1的节点的请求可以被取消或者忽略。当n=2的时候，这就是back request模式。 这样可以用资源来换取latency的提升。
//
//需要注意的是，当收到任意一个信号的时候，其它信号都被忽略。如果用channel来实现，只要从任意一个channel中接收到一个数据，那么所有的channel都可以被关闭了(依照你的实现，但是输出的channel肯定会被关闭)。
//
//有三种实现的方式: goroutine、reflect和递归。
//---------------------------------------------------------------------------------------------------------------------------------------------
// 1、Goroutine方式
//
//or函数可以处理n个channel,它为每个channel启动一个goroutine，只要任意一个goroutine从channel读取到数据，输出的channel就被关闭掉了。
//
//为了避免并发关闭输出channel的问题，关闭操作只执行一次。
func or1(chans ...<-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		var once sync.Once
		for _, c := range chans {
			go func(c <-chan interface{}) {
				select {
				case <-c:
					once.Do(func() { close(out) })
				case <-out:
				}
			}(c)
		}
	}()
	return out
}

//2、Reflect方式
//Go的反射库针对select语句有专门的数据(reflect.SelectCase)和函数(reflect.Select)处理。
//所以我们可以利用反射“随机”地从一组可选的channel中接收数据，并关闭输出channel。

func or2(channels ...<-chan interface{}) <-chan interface{} {
	switch len(channels) {
	case 0:
		return nil
	case 1:
		return channels[0]
	}
	orDone := make(chan interface{})
	go func() {
		defer close(orDone)
		var cases []reflect.SelectCase
		for _, c := range channels {
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(c),
			})
		}
		reflect.Select(cases)
	}()
	return orDone
}

//3、递归方式
//
//递归方式一向是比较开脑洞的实现，下面的方式就是分而治之的方式，逐步合并channel，最终返回一个channel。
//
//在后面的扇入(合并)模式中，我们还是会使用相同样的递归模式来合并多个输入channel，根据 justforfun 的测试结果，这种递归的方式要比goroutine、Reflect更有效。
//
func or3(channels ...<-chan interface{}) <-chan interface{} {
	switch len(channels) {
	case 0:
		return nil
	case 1:
		return channels[0]
	}
	orDone := make(chan interface{})
	go func() {
		defer close(orDone)
		switch len(channels) {
		case 2:
			select {
			case <-channels[0]:
			case <-channels[1]:
			}
		default:
			m := len(channels) / 2
			select {
			case <-or3(channels[:m]...):
			case <-or3(channels[m:]...):
			}
		}
	}()
	return orDone
}