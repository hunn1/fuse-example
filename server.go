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
		Timeout:                int(3 * time.Second),
		MaxConcurrentRequests:  10,
		SleepWindow:            30000,
		RequestVolumeThreshold: 20,
		ErrorPercentThreshold:  30,
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
	go http.ListenAndServe(net.JoinHostPort("", "81"), hystrixStreamHandler)
	http.ListenAndServe(":8090", &Handle{})

}
