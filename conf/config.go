package conf

import (
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
	"impala_metrics/metrics"
	"log"
	"os"
	"sync/atomic"
)

type MetricConfig struct {
	Name string `yaml:"name"`
	Help string `yaml:"help"`
}

type Config struct {
	Servers         []string                `yaml:"servers"`
	IntervalSeconds int                     `yaml:"intervalSeconds"`
	Port            int                     `yaml:"port"`
	NumWorkers      int                     `yaml:"numWorkers"`
	Metrics         map[string]MetricConfig `yaml:"metrics"`
}

// ReadConfig 读取配置文件
func ReadConfig(configFilePath string, logger *log.Logger, safeConfig *atomic.Value, prometheusMetrics *atomic.Value, lastPrometheusMetrics *atomic.Value) error {
	initConfig, err := loadConfig(configFilePath, logger)
	if err != nil {
		logger.Fatalf("解析YAML时发生错误: %v", err)
		return err
	}

	if lastPrometheusMetrics.Load() != nil {
		lastPrometheusMetricsNames := lastPrometheusMetrics.Load().([]*prometheus.GaugeVec)
		for _, v := range lastPrometheusMetricsNames {
			logger.Printf("start to unregister:%v\n", v)
			if !prometheus.Unregister(v) {
				logger.Fatalf("unregister failed with %v", err)
				return err
			}
		}
	}
	// 重置lastPrometheusMetricsNames
	lastPrometheusMetricsNames := []*prometheus.GaugeVec{}
	allPrometheusMetrics := make(map[string]*prometheus.GaugeVec)
	for key, mc := range initConfig.Metrics {
		gaugeVec := metrics.CreateGaugeVec(mc.Name, mc.Help)
		if gaugeVec == nil {
			logger.Fatalf("Create GaugeVec failed with %v", err)
			return err
		}
		allPrometheusMetrics[key] = gaugeVec
		lastPrometheusMetricsNames = append(lastPrometheusMetricsNames, gaugeVec)
		prometheus.MustRegister(gaugeVec)
	}
	logger.Printf("metricNames: %v\n", initConfig.Metrics)
	logger.Printf("allPrometheusMetrics: %v\n", allPrometheusMetrics)

	safeConfig.Store(initConfig)
	prometheusMetrics.Store(allPrometheusMetrics)
	lastPrometheusMetrics.Store(lastPrometheusMetricsNames)

	return err
}

// loadConfig 加载配置文件
func loadConfig(configFilePath string, logger *log.Logger) (*Config, error) {
	yamlFile, err := os.ReadFile(configFilePath)
	if err != nil {
		logger.Fatalf("读取文件时发生错误: %v", err)
	}
	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
