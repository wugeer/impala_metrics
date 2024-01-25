# impala Metrics Collector
## 介绍
Impala Metrics Collector 是一个用于收集和处理 Apache Impala 相关度量指标的工具。它旨在简化监控和分析 Impala 性能的过程，提供了一个可配置的方式来获取、存储和可能的情况下分析关键性能指标。
本工具抓取的impala 25000 webui页面的metrics数据，将其转换为prometheus格式, 目前label只有ip和queue两个，然后通过http接口暴露出来，供prometheus采集, 在25000/metrics上的指标支持在配置文件热加载的情况下动态添加(注意该配置项如何配置)

## 特性
- 指标收集: 自动收集来自 Impala 实例的关键性能指标。
- 配置驱动: 通过 config.yml 文件轻松配置指标和收集频率, 支持热加载, 自动识别配置文件变更，更新配置信息。
- 并行处理: 利用 worker 模式并发处理数据收集。
- 日志记录: 提供详细的日志记录，以便于问题跟踪和调试。
- 信号处理: 支持优雅地处理操作系统信号，如中断和终止。

## 目录结构
```bash
impala_metrics/
├── conf/                # 配置处理相关代码
│   └── config.go
├── logger/              # 日志处理相关代码
│   └── logger.go
├── metrics/             # 指标收集相关代码
│   └── metrics.go
├── utils/               # 工具代码，如信号处理和配置监控
│   ├── signal_handler.go
│   └── watch_config.go
├── worker/              # 并行工作处理逻辑
│   └── worker.go
├── config.yml           # 指标收集配置文件
├── main.go              # 程序入口
└── README.md            # 项目文档
```

## 快速开始
### 安装
确保您的系统已安装 Go (版本 1.x)。
克隆项目到本地：
```bash
git clone [repository-url]
```
进入项目目录：
```bash
cd impala_metrics
```
安装依赖：
```bash
go get -u github.com/prometheus/client_golang/prometheus
go get -u gopkg.in/yaml.v2
go get -u github.com/PuerkitoBio/goquery
```
## 配置
编辑 config.yml 文件以适应您的环境和需求。您可以配置的选项包括：(加粗的是当前不支持热加载的)
- Impala 服务器地址
- 指标收集间隔
- **node exporter http 端口**
- **Worker 数量**
- 需要采集的指标

这里对配置文件中的需要采集的指标做一下说明：
```bash
  admission-controller.local-mem-admitted:             # 这个是impala的指标名称，在25000/metrics页面根据这个名称查找对应的值
    name: "admission_controller_local_mem_admitted"    # 存储到prometheus中的指标名称
    help: "Admission controller local memory admitted" # 存储到prometheus中的指标描述
```
对应的prometheus指标样例为
```bash
admision_controller_local_mem_admitted{ip="host1",queue="default"} 0
```

## 运行
在项目根目录下运行：
```bash
go run main.go
```
或者编译后运行：
```bash
CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o impalaMetrics main.go
chmod +x impalaMetrics
./impalaMetrics --config-file config.yml
```

## 贡献
我们欢迎任何形式的贡献，无论是新功能，文档改进还是问题报告。请使用 GitHub 的 Issues 和 Pull Requests 功能来参与贡献。

## 许可
此项目使用 MIT 许可证。