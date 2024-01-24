package worker

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"impala_metrics/metrics"
	"log"
	"sync"
	"sync/atomic"
)

// Job 表示一个工作项，这里我们定义为服务器的server地址
type Job struct {
	Server string
}

// Worker 函数表示每个工作者如何处理工作
func Worker(id int, jobs <-chan Job, wg *sync.WaitGroup, logger *log.Logger, prometheusMetrics *atomic.Value) {
	logger.Printf("worker [%d] started\n", id)
	for j := range jobs {
		prometheusMetricsNames := prometheusMetrics.Load().(map[string]*prometheus.GaugeVec)
		var metricNames = []string{}

		for key, _ := range prometheusMetricsNames {
			metricNames = append(metricNames, key)
		}
		processMetrics(j.Server, metricNames, prometheusMetricsNames, logger)
		wg.Done()
	}
}

// processMetrics 处理指标抓取和更新
func processMetrics(server string, metricNames []string, prometheusMetricsNames map[string]*prometheus.GaugeVec, logger *log.Logger) {
	url := fmt.Sprintf("http://%v:25000/metrics", server)
	logger.Printf("start to scrapy the %v metrics of server: %v\n", metricNames, url)
	value, err := metrics.GetMetricValue(url, metricNames, logger)
	if err != nil {
		log.Fatal(err)
	}
	// 遍历输出的map类型指标
	for k, v := range value {
		for _, v1 := range v {
			prometheusMetricsNames[k].WithLabelValues(server, v1.Queue).Set(v1.Value)
		}
	}
}
