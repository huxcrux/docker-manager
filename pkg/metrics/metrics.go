package metrics

import (
	"github.com/docker/docker/api/types"
	"github.com/prometheus/client_golang/prometheus"
)

// DockerMetrics holds Prometheus metrics
type DockerMetrics struct {
	CPUUsageTotal      *prometheus.GaugeVec
	MemoryUsage        *prometheus.GaugeVec
	MemoryMaxUsage     *prometheus.GaugeVec
	MemoryLimit        *prometheus.GaugeVec
	MemoryCache        *prometheus.GaugeVec
	MemoryRSS          *prometheus.GaugeVec
	MemoryUsageOverall *prometheus.GaugeVec
	NetworkRxBytes     *prometheus.GaugeVec
	NetworkTxBytes     *prometheus.GaugeVec
	BlockIoReadBytes   *prometheus.GaugeVec
	BlockIoWriteBytes  *prometheus.GaugeVec
}

// NewDockerMetrics initializes and registers Prometheus metrics
func NewDockerMetrics() *DockerMetrics {
	dm := &DockerMetrics{
		CPUUsageTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_cpu_usage_total",
				Help: "Total CPU usage of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		MemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_memory_usage",
				Help: "Memory usage of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		MemoryMaxUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_memory_max_usage",
				Help: "Maximum memory usage of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		MemoryLimit: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_memory_limit",
				Help: "Memory limit of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		MemoryCache: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_memory_cache",
				Help: "Cache memory usage of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		MemoryRSS: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_memory_rss",
				Help: "RSS memory usage of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		MemoryUsageOverall: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_memory_usage_overall",
				Help: "Overall memory usage of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		NetworkRxBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_network_rx_bytes",
				Help: "Network received bytes of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		NetworkTxBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_network_tx_bytes",
				Help: "Network transmitted bytes of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		BlockIoReadBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_block_io_read_bytes",
				Help: "Block IO read bytes of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
		BlockIoWriteBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "docker_block_io_write_bytes",
				Help: "Block IO write bytes of Docker containers",
			},
			[]string{"container_id", "container_name"},
		),
	}

	// Register all metrics with Prometheus
	prometheus.MustRegister(dm.CPUUsageTotal)
	prometheus.MustRegister(dm.MemoryUsage)
	prometheus.MustRegister(dm.MemoryMaxUsage)
	prometheus.MustRegister(dm.MemoryLimit)
	prometheus.MustRegister(dm.MemoryCache)
	prometheus.MustRegister(dm.MemoryRSS)
	prometheus.MustRegister(dm.MemoryUsageOverall)
	prometheus.MustRegister(dm.NetworkRxBytes)
	prometheus.MustRegister(dm.NetworkTxBytes)
	prometheus.MustRegister(dm.BlockIoReadBytes)
	prometheus.MustRegister(dm.BlockIoWriteBytes)

	return dm
}

// UpdateMetrics updates Prometheus metrics with values from types.StatsJSON
func (dm *DockerMetrics) UpdateMetrics(stats types.StatsJSON) {
	containerID := stats.ID
	containerName := stats.Name

	// CPU usage calculation
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cpuPercent := (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0

	// Memory usage
	memoryUsage := float64(stats.MemoryStats.Usage)
	memoryMaxUsage := float64(stats.MemoryStats.MaxUsage)
	memoryLimit := float64(stats.MemoryStats.Limit)
	memoryCache := float64(stats.MemoryStats.Stats["cache"])
	memoryRSS := float64(stats.MemoryStats.Stats["rss"])
	overallMemoryUsage := memoryUsage - memoryCache

	// Network I/O
	var rxBytes, txBytes uint64
	for _, v := range stats.Networks {
		rxBytes += v.RxBytes
		txBytes += v.TxBytes
	}

	// Block I/O
	var blkRead, blkWrite uint64
	for _, bio := range stats.BlkioStats.IoServiceBytesRecursive {
		if bio.Op == "read" {
			blkRead += bio.Value
		} else if bio.Op == "write" {
			blkWrite += bio.Value
		}
	}

	// Set Prometheus metrics
	dm.CPUUsageTotal.WithLabelValues(containerID, containerName).Set(cpuPercent)
	dm.MemoryUsage.WithLabelValues(containerID, containerName).Set(memoryUsage)
	dm.MemoryMaxUsage.WithLabelValues(containerID, containerName).Set(memoryMaxUsage)
	dm.MemoryLimit.WithLabelValues(containerID, containerName).Set(memoryLimit)
	dm.MemoryCache.WithLabelValues(containerID, containerName).Set(memoryCache)
	dm.MemoryRSS.WithLabelValues(containerID, containerName).Set(memoryRSS)
	dm.MemoryUsageOverall.WithLabelValues(containerID, containerName).Set(overallMemoryUsage)
	dm.NetworkRxBytes.WithLabelValues(containerID, containerName).Set(float64(rxBytes))
	dm.NetworkTxBytes.WithLabelValues(containerID, containerName).Set(float64(txBytes))
	dm.BlockIoReadBytes.WithLabelValues(containerID, containerName).Set(float64(blkRead))
	dm.BlockIoWriteBytes.WithLabelValues(containerID, containerName).Set(float64(blkWrite))
}
