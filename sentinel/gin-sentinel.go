package main

import (
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/gin-gonic/gin"
	sentinelPlugin "github.com/sentinel-group/sentinel-go-adapters/gin"
	"log"
)

func init() {
	//初始化 sentinel
	conf := config.NewDefaultConfig()
	// "cb-integration-normal"
	conf.Sentinel.Log.Logger = logging.NewConsoleLogger()
	conf.Sentinel.Log.Dir = "."
	err := sentinel.InitWithConfig(conf)
	if err != nil {
		log.Fatal("err", err)
	}

	//go语言实现暂未发现appLimit即限制客户端每秒请求次数
	if _, err := flow.LoadRules([]*flow.Rule{
		{
			Resource:  "abc", // 资源名，即规则的作用目标。
			Threshold: 0.1,
			// 表示流控阈值；如果字段 StatIntervalInMs 是1000(也就是1秒)，
			// 那么Threshold就表示QPS，流量控制器也就会依据资源的QPS来做流控。

			//Resource:               "GET:/health",
			//Threshold:              1.0,
			//TokenCalculateStrategy: flow.Direct,
			//ControlBehavior:        flow.Reject,
			//StatIntervalInMs:       1000,

			//MetricType:             flow.QPS,
			//Count:                  1,
			//TokenCalculateStrategy: flow.Direct,
			//ControlBehavior:        flow.Reject,
		},
	}); err != nil {
		log.Fatalf("Unexpected error: %+v", err)
		return
	}

}

func main() {
	r := gin.Default()
	// Sentinel 会对每个 API route 进行统计，资源名称类似于 GET:/foo/:id
	// 默认的限流处理逻辑是返回 429 (Too Many Requests) 错误码，支持配置自定义的 fallback 逻辑
	r.Use(sentinelPlugin.SentinelMiddleware(
		sentinelPlugin.WithResourceExtractor(func(ctx *gin.Context) string {
			return ctx.GetHeader("X-Real-IP")
		}),
		// customize block fallback if required
		// abort with status 429 by default
		sentinelPlugin.WithBlockFallback(func(ctx *gin.Context) {
			ctx.AbortWithStatusJSON(400, map[string]interface{}{
				"err":  "too many request; the quota used up",
				"code": 10222,
			})
		}),
	))

	_, b := sentinel.Entry("GET:/health", sentinel.WithTrafficType(base.Inbound))
	if b != nil {
		// Blocked. We could get the block reason from the BlockError.
		//time.Sleep(time.Duration(rand.Uint64()%10) * time.Millisecond)
		r.GET("/health", func(context *gin.Context) {
			context.JSON(200, map[string]string{"status": b.BlockMsg()})
		})
	} else {
		r.GET("/health", func(context *gin.Context) {
			context.JSON(200, map[string]string{"status": "ok"})
		})
	}

	_ = r.Run("0.0.0.0:8088")
}
