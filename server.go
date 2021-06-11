package main

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"log"
	"net"
	"net/http"
	"time"
)

type Handle struct{}

func (h *Handle) ServeHTTP(r http.ResponseWriter, request *http.Request) {
	h.Common(r, request)
}

func (h *Handle) Common(r http.ResponseWriter, request *http.Request) {
	hystrix.ConfigureCommand("mycommand", hystrix.CommandConfig{
		Timeout: int(3 * time.Second),
		// command的最大并发量
		// 如果达到这个最大并发限制，则后续请求将被拒绝。
		// 当你选择一个信号量时，你使用的逻辑基本上和你选择线程池中添加多少个线程相同，但是信号量的开销要小得多，通常执行速度要快得多（亚毫秒） ，否则你会使用线程。
		MaxConcurrentRequests: 10,
		// 该属性设置跳闸后的时间量，拒绝请求，然后再次尝试确定电路是否应再次闭合。
		// 过多长时间，熔断器再次检测是否开启。单位毫秒
		SleepWindow: 30000,
		// 例如，如果值是20，那么如果在滚动窗口中接收到19个请求（例如10秒的窗口），则即使所有19个请求都失败，电路也不会跳闸。
		RequestVolumeThreshold: 20,
		// 错误率 请求数量大于等于RequestVolumeThreshold并且错误率到达这个百分比后就会启动
		ErrorPercentThreshold: 30,
	})
	msg := "success"

	_ = hystrix.Do("mycommand", func() error {
		_, err := http.Get("https://www.baidu.com")
		if err != nil {
			fmt.Printf("请求失败:%v\n", err)
			return err
		}
		return nil
	}, func(err error) error {
		//fmt.Printf("handle  error:%v\n", err)
		//msg = "error"
		//加入自动降级处理，如获取缓存数据等
		switch err {
		case hystrix.ErrCircuitOpen:
			fmt.Println("circuit error:" + err.Error())
			msg = "ErrCircuitOpen"
		case hystrix.ErrMaxConcurrency:
			fmt.Println("circuit error:" + err.Error())
			msg = "ErrMaxConcurrency"
		default:
			fmt.Println("other error:" + err.Error())
			msg = "circuit error"
		}

		time.Sleep(1 * time.Second)
		log.Println("sleep 1 second")

		return nil
	})
	r.Write([]byte(msg))
}

func main() {
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	// 熔断器的输出流
	// http://localhost:81
	go http.ListenAndServe(net.JoinHostPort("", "81"), hystrixStreamHandler)
	// 网站服务
	// http://localhost:8090
	http.ListenAndServe(":8090", &Handle{})

}
