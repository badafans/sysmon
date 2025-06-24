package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SystemStats 系统状态结构体
type SystemStats struct {
	RunTime            string `json:"run_time"`
	Last1              string `json:"last1"`
	Last5              string `json:"last5"`
	Last15             string `json:"last15"`
	CPUUsage           string `json:"cpu_usage"`
	CPUTemp            string `json:"cpu_temp"`
	MemTotalSpace      string `json:"mem_total_space"`
	MemUsedSpace       string `json:"mem_used_space"`
	MemFreeSpace       string `json:"mem_free_space"`
	MemUsage           string `json:"mem_usage"`
	SwapTotalSpace     string `json:"swap_total_space"`
	SwapUsedSpace      string `json:"swap_used_space"`
	SwapFreeSpace      string `json:"swap_free_space"`
	DiskTotalSpace     string `json:"disk_total_space"`
	DiskUsedSpace      string `json:"disk_used_space"`
	DiskAvailableSpace string `json:"disk_available_space"`
	DiskUsage          string `json:"disk_usage"`
	ReceiveSpeed       string `json:"receive_speed"`
	TransmitSpeed      string `json:"transmit_speed"`
	ReceiveTotal       string `json:"receive_total"`
	TransmitTotal      string `json:"transmit_total"`
	LatestTime         string `json:"lastest_time"`
}

// Config 简化配置结构体
type Config struct {
	Interface string
	Port      int
	Interval  time.Duration
}

// Monitor 系统监控器
type Monitor struct {
	config      Config
	prevNetRx   uint64
	prevNetTx   uint64
	prevCPUStat CPUStat
}

// CPUStat CPU统计信息
type CPUStat struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	Iowait  uint64
	Irq     uint64
	Softirq uint64
}

// EnhancedMonitor 增强监控器
type EnhancedMonitor struct {
	config       Config
	monitor      *Monitor
	currentStats SystemStats
}

var htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>系统监控面板</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container { 
            max-width: 1400px; 
            margin: 0 auto; 
        }
        .header { 
            text-align: center; 
            color: white; 
            margin-bottom: 40px;
            text-shadow: 0 2px 4px rgba(0,0,0,0.3);
        }
        .header h1 {
            font-size: 2.5rem;
            font-weight: 300;
            margin-bottom: 10px;
        }
        .stats-grid { 
            display: grid; 
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); 
            gap: 25px; 
        }
        .stat-card { 
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            padding: 25px; 
            border-radius: 16px; 
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
            border: 1px solid rgba(255,255,255,0.2);
            transition: all 0.3s ease;
            position: relative;
            overflow: hidden;
        }
        .stat-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 12px 40px rgba(0,0,0,0.15);
        }
        .stat-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 4px;
            background: linear-gradient(90deg, #667eea, #764ba2);
        }
        .stat-title { 
            font-size: 20px; 
            font-weight: 600; 
            color: #2c3e50; 
            margin-bottom: 20px; 
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .stat-title .icon {
            font-size: 24px;
        }
        .stat-item { 
            display: flex; 
            justify-content: space-between; 
            align-items: center;
            margin: 12px 0; 
            padding: 8px 0;
            border-bottom: 1px solid rgba(0,0,0,0.05);
        }
        .stat-item:last-child {
            border-bottom: none;
        }
        .stat-label { 
            color: #7f8c8d; 
            font-weight: 500;
        }
        .stat-value { 
            font-weight: 700; 
            color: #2c3e50;
            font-size: 16px;
            transition: all 0.3s ease;
        }
        .progress-bar { 
            width: 100%; 
            height: 8px; 
            background-color: rgba(0,0,0,0.1); 
            border-radius: 4px; 
            overflow: hidden; 
            margin: 10px 0;
            position: relative;
        }
        .progress-fill { 
            height: 100%; 
            transition: width 0.5s ease;
            border-radius: 4px;
            position: relative;
        }
        .progress-fill::after {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: linear-gradient(90deg, transparent, rgba(255,255,255,0.3), transparent);
            animation: shimmer 2s infinite;
        }
        @keyframes shimmer {
            0% { transform: translateX(-100%); }
            100% { transform: translateX(100%); }
        }
        .cpu-usage { background: linear-gradient(90deg, #ff6b6b, #ff8e8e); }
        .memory-usage { background: linear-gradient(90deg, #4ecdc4, #7ed6cc); }
        .disk-usage { background: linear-gradient(90deg, #45b7d1, #74c7e3); }
        .system-card .stat-title { color: #3498db; }
        .cpu-card .stat-title { color: #e74c3c; }
        .memory-card .stat-title { color: #1abc9c; }
        .disk-card .stat-title { color: #3498db; }
        .network-card .stat-title { color: #9b59b6; }
        .swap-card .stat-title { color: #f39c12; }
        .update-time { 
            text-align: center; 
            margin-top: 30px; 
            color: rgba(255,255,255,0.9); 
            font-size: 14px;
            background: rgba(255,255,255,0.1);
            padding: 15px;
            border-radius: 10px;
            backdrop-filter: blur(10px);
            transition: opacity 0.3s ease;
        }
        .network-speed {
            display: flex;
            gap: 15px;
            margin: 15px 0;
        }
        .speed-item {
            flex: 1;
            text-align: center;
            padding: 10px;
            background: rgba(155, 89, 182, 0.1);
            border-radius: 8px;
            border: 1px solid rgba(155, 89, 182, 0.2);
        }
        .speed-value {
            font-size: 18px;
            font-weight: bold;
            color: #9b59b6;
            display: block;
        }
        .speed-label {
            font-size: 12px;
            color: #7f8c8d;
            margin-top: 5px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🖥️ 系统监控面板</h1>
            <p>实时系统性能监控</p>
        </div>
        
        <div class="stats-grid">
            <!-- 系统信息 -->
            <div class="stat-card system-card">
                <div class="stat-title">
                    <span class="icon">🖥️</span>
                    系统信息
                </div>
                <div class="stat-item">
                    <span class="stat-label">运行时间:</span>
                    <span class="stat-value">{{.Stats.RunTime}}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">平均负载:</span>
                    <span class="stat-value">{{.Stats.Last1}} {{.Stats.Last5}} {{.Stats.Last15}}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">CPU温度:</span>
                    <span class="stat-value">{{.Stats.CPUTemp}}</span>
                </div>
            </div>

            <!-- CPU使用率 -->
            <div class="stat-card cpu-card">
                <div class="stat-title">
                    <span class="icon">🔥</span>
                    CPU 使用率
                </div>
                <div class="stat-item">
                    <span class="stat-label">当前使用率:</span>
                    <span class="stat-value">{{.Stats.CPUUsage}}%</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill cpu-usage" style="width: {{.Stats.CPUUsage}}%"></div>
                </div>
            </div>

            <!-- 内存信息 -->
            <div class="stat-card memory-card">
                <div class="stat-title">
                    <span class="icon">🧠</span>
                    内存信息
                </div>
                <div class="stat-item">
                    <span class="stat-label">总容量:</span>
                    <span class="stat-value">{{.Stats.MemTotalSpace}} MB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">已使用:</span>
                    <span class="stat-value">{{.Stats.MemUsedSpace}} MB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">可用:</span>
                    <span class="stat-value">{{.Stats.MemFreeSpace}} MB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">使用率:</span>
                    <span class="stat-value">{{.Stats.MemUsage}}%</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill memory-usage" style="width: {{.Stats.MemUsage}}%"></div>
                </div>
            </div>

            <!-- 磁盘信息 -->
            <div class="stat-card disk-card">
                <div class="stat-title">
                    <span class="icon">💾</span>
                    磁盘信息
                </div>
                <div class="stat-item">
                    <span class="stat-label">总容量:</span>
                    <span class="stat-value">{{.Stats.DiskTotalSpace}} GB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">已使用:</span>
                    <span class="stat-value">{{.Stats.DiskUsedSpace}} GB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">可用:</span>
                    <span class="stat-value">{{.Stats.DiskAvailableSpace}} GB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">使用率:</span>
                    <span class="stat-value">{{.Stats.DiskUsage}}%</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill disk-usage" style="width: {{.Stats.DiskUsage}}%"></div>
                </div>
            </div>

            <!-- 网络信息 -->
            <div class="stat-card network-card">
                <div class="stat-title">
                    <span class="icon">🌐</span>
                    网络信息
                    <select id="interface-selector" style="margin-left: 10px; padding: 2px 5px; border-radius: 3px; border: 1px solid #ddd; font-size: 12px;">
                        <option value="">加载中...</option>
                    </select>
                </div>
                <div class="network-speed">
                    <div class="speed-item">
                        <span class="speed-value">{{.Stats.ReceiveSpeed}}</span>
                        <div class="speed-label">接收速率 (kB/s)</div>
                    </div>
                    <div class="speed-item">
                        <span class="speed-value">{{.Stats.TransmitSpeed}}</span>
                        <div class="speed-label">发送速率 (kB/s)</div>
                    </div>
                </div>
                <div class="stat-item">
                    <span class="stat-label">累计接收:</span>
                    <span class="stat-value">{{.Stats.ReceiveTotal}} GB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">累计发送:</span>
                    <span class="stat-value">{{.Stats.TransmitTotal}} GB</span>
                </div>
            </div>

            <!-- SWAP信息 -->
            <div class="stat-card swap-card">
                <div class="stat-title">
                    <span class="icon">🔄</span>
                    SWAP 信息
                </div>
                <div class="stat-item">
                    <span class="stat-label">总容量:</span>
                    <span class="stat-value">{{.Stats.SwapTotalSpace}} MB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">已使用:</span>
                    <span class="stat-value">{{.Stats.SwapUsedSpace}} MB</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">空闲:</span>
                    <span class="stat-value">{{.Stats.SwapFreeSpace}} MB</span>
                </div>
            </div>
        </div>
        
        <div class="update-time">
            📊 最后更新: <span id="last-update">{{.Stats.LatestTime}}</span>
        </div>
    </div>
    
    <script>
        let updateInterval = {{.Interval}} * 1000; // 转换为毫秒
        
        function updateStats() {
            fetch('/api/stats')
                .then(response => {
                    if (!response.ok) {
                        throw new Error('网络请求失败');
                    }
                    return response.json();
                })
                .then(data => {
                    // 更新系统信息
                    document.querySelector('.system-card .stat-item:nth-child(2) .stat-value').textContent = data.run_time;
                    document.querySelector('.system-card .stat-item:nth-child(3) .stat-value').textContent = data.last1 + ' ' + data.last5 + ' ' + data.last15;
                    document.querySelector('.system-card .stat-item:nth-child(4) .stat-value').textContent = data.cpu_temp;
                    
                    // 更新CPU使用率
                    document.querySelector('.cpu-card .stat-value').textContent = data.cpu_usage + '%';
                    const cpuValue = parseFloat(data.cpu_usage);
                    document.querySelector('.cpu-usage').style.width = cpuValue + '%';
                    
                    // 更新内存信息
                    const memoryItems = document.querySelectorAll('.memory-card .stat-item .stat-value');
                    memoryItems[0].textContent = data.mem_total_space + ' MB';
                    memoryItems[1].textContent = data.mem_used_space + ' MB';
                    memoryItems[2].textContent = data.mem_free_space + ' MB';
                    memoryItems[3].textContent = data.mem_usage + '%';
                    const memValue = parseFloat(data.mem_usage);
                    document.querySelector('.memory-usage').style.width = memValue + '%';
                    
                    // 更新磁盘信息
                    const diskItems = document.querySelectorAll('.disk-card .stat-item .stat-value');
                    diskItems[0].textContent = data.disk_total_space + ' GB';
                    diskItems[1].textContent = data.disk_used_space + ' GB';
                    diskItems[2].textContent = data.disk_available_space + ' GB';
                    diskItems[3].textContent = data.disk_usage + '%';
                    const diskValue = parseFloat(data.disk_usage);
                    document.querySelector('.disk-usage').style.width = diskValue + '%';
                    
                    // 更新网络信息
                    const speedValues = document.querySelectorAll('.network-card .speed-value');
                    speedValues[0].textContent = data.receive_speed;
                    speedValues[1].textContent = data.transmit_speed;
                    const networkItems = document.querySelectorAll('.network-card .stat-item .stat-value');
                    networkItems[0].textContent = data.receive_total + ' GB';
                    networkItems[1].textContent = data.transmit_total + ' GB';
                    
                    // 更新SWAP信息
                    const swapItems = document.querySelectorAll('.swap-card .stat-item .stat-value');
                    swapItems[0].textContent = data.swap_total_space + ' MB';
                    swapItems[1].textContent = data.swap_used_space + ' MB';
                    swapItems[2].textContent = data.swap_free_space + ' MB';
                    
                    // 更新时间戳
                    document.getElementById('last-update').textContent = data.lastest_time;
                    
                    // 添加轻微的更新指示效果（避免抖动）
                    document.querySelector('.update-time').style.opacity = '0.7';
                    setTimeout(() => {
                        document.querySelector('.update-time').style.opacity = '1';
                    }, 100);
                })
                .catch(error => {
                    console.error('更新数据失败:', error);
                    // 显示错误提示
                    document.getElementById('last-update').textContent = '更新失败 - ' + new Date().toLocaleTimeString();
                });
        }
        
        // 加载网络接口列表
        function loadInterfaces() {
            fetch('/api/interfaces')
                .then(response => response.json())
                .then(data => {
                    const selector = document.getElementById('interface-selector');
                    selector.innerHTML = '';
                    data.interfaces.forEach(intf => {
                        const option = document.createElement('option');
                        option.value = intf;
                        option.textContent = intf;
                        if (intf === data.current) {
                            option.selected = true;
                        }
                        selector.appendChild(option);
                    });
                })
                .catch(error => {
                    console.error('加载网络接口失败:', error);
                    const selector = document.getElementById('interface-selector');
                    selector.innerHTML = '<option value="">加载失败</option>';
                });
        }

        // 切换网络接口
        function switchInterface(interfaceName) {
            fetch('/api/switch-interface', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ interface: interfaceName })
            })
            .then(response => response.json())
            .then(data => {
                if (data.status === 'success') {
                    console.log('网络接口切换成功:', interfaceName);
                    // 立即更新一次统计数据
                    updateStats();
                } else {
                    console.error('网络接口切换失败');
                }
            })
            .catch(error => {
                console.error('网络接口切换失败:', error);
            });
        }

        // 页面加载完成后开始定时更新
        document.addEventListener('DOMContentLoaded', function() {
            loadInterfaces();
            // 立即执行一次更新
            setTimeout(updateStats, 1000);
            // 设置定时更新
            setInterval(updateStats, updateInterval);
            
            // 绑定接口选择器事件
            document.getElementById('interface-selector').addEventListener('change', function() {
                const selectedInterface = this.value;
                if (selectedInterface) {
                    switchInterface(selectedInterface);
                }
            });
        });
        
        // 添加页面可见性检测，当页面不可见时停止更新
        let updateTimer;
        document.addEventListener('visibilitychange', function() {
            if (document.hidden) {
                clearInterval(updateTimer);
            } else {
                updateTimer = setInterval(updateStats, updateInterval);
            }
        });
    </script>
</body>
</html>
`

func main() {
	// 解析命令行参数
	var (
		port = flag.Int("port", 8080, "Web服务器端口")
	)
	flag.Parse()

	// 创建临时监控器来获取可用网卡
	tempMonitor := &Monitor{}
	interfaces := tempMonitor.getAvailableInterfaces()
	defaultInterface := ""

	// 自动选择第一个可用网卡
	if len(interfaces) > 0 {
		defaultInterface = interfaces[0]
		// 保存自动选择的网卡到内存
		setSelectedInterface(defaultInterface)
	}

	// 创建配置
	config := Config{
		Interface: defaultInterface,
		Port:      *port,
		Interval:  1 * time.Second,
	}

	// 创建增强监控器
	enhancedMonitor := &EnhancedMonitor{
		config: config,
	}

	// 创建基础监控器
	enhancedMonitor.monitor = &Monitor{
		config: config,
	}

	// 初始化统计
	enhancedMonitor.monitor.initStats()

	// 启动Web服务器
	go enhancedMonitor.startWebServer()

	// 开始监控循环
	for {
		stats := enhancedMonitor.monitor.collectStats()
		enhancedMonitor.currentStats = stats
		time.Sleep(config.Interval)
	}
}

// startWebServer 启动Web服务器
func (em *EnhancedMonitor) startWebServer() {
	tmpl := template.Must(template.New("monitor").Parse(htmlTemplate))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Stats    SystemStats
			Interval int
		}{
			Stats:    em.currentStats,
			Interval: int(em.config.Interval.Seconds()),
		}
		tmpl.Execute(w, data)
	})

	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(em.currentStats)
	})

	// 获取可用网络接口列表
	http.HandleFunc("/api/interfaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		interfaces := em.monitor.getAvailableInterfaces()
		response := map[string]interface{}{
			"interfaces": interfaces,
			"current":    getSelectedInterface(),
		}
		json.NewEncoder(w).Encode(response)
	})

	// 切换网络接口
	http.HandleFunc("/api/switch-interface", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Interface string `json:"interface"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		// 验证接口是否存在
		interfaces := em.monitor.getAvailableInterfaces()
		valid := false
		for _, intf := range interfaces {
			if intf == req.Interface {
				valid = true
				break
			}
		}
		if !valid {
			http.Error(w, "Interface not found", http.StatusBadRequest)
			return
		}
		// 切换接口
		em.config.Interface = req.Interface
		em.monitor.config.Interface = req.Interface
		// 保存用户选择到内存
		setSelectedInterface(req.Interface)
		// 重置网络统计
		em.monitor.prevNetRx, em.monitor.prevNetTx = em.monitor.getNetworkStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	addr := fmt.Sprintf(":%d", em.config.Port)
	log.Printf("Web服务器启动在 %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// getAvailableInterfaces 获取可用的网络接口列表
func (m *Monitor) getAvailableInterfaces() []string {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return []string{}
	}

	lines := strings.Split(string(data), "\n")
	var interfaces []string
	for i := 2; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		parts := strings.Split(lines[i], ":")
		if len(parts) >= 2 {
			intf := strings.TrimSpace(parts[0])
			if intf != "lo" {
				interfaces = append(interfaces, intf)
			}
		}
	}
	return interfaces
}

// 当前选择的网卡接口（内存中保存）
var currentSelectedInterface string

// setSelectedInterface 设置当前选择的网卡接口
func setSelectedInterface(interfaceName string) {
	currentSelectedInterface = interfaceName
}

// getSelectedInterface 获取当前选择的网卡接口
func getSelectedInterface() string {
	return currentSelectedInterface
}

// initStats 初始化统计数据
func (m *Monitor) initStats() {
	m.prevNetRx, m.prevNetTx = m.getNetworkStats()
	m.prevCPUStat = m.getCPUStats()
}

// collectStats 收集系统统计信息
func (m *Monitor) collectStats() SystemStats {
	// 获取当前网络和CPU统计
	currNetRx, currNetTx := m.getNetworkStats()
	currCPUStat := m.getCPUStats()

	// 计算网络速度
	receiveSpeed := float64(currNetRx-m.prevNetRx) / 1024 / m.config.Interval.Seconds()
	transmitSpeed := float64(currNetTx-m.prevNetTx) / 1024 / m.config.Interval.Seconds()

	// 计算CPU使用率
	cpuUsage := m.calculateCPUUsage(m.prevCPUStat, currCPUStat)

	// 更新前一次的统计数据
	m.prevNetRx, m.prevNetTx = currNetRx, currNetTx
	m.prevCPUStat = currCPUStat

	// 获取其他系统信息
	uptime := m.getUptime()
	loadAvg := m.getLoadAverage()
	cpuTemp := m.getCPUTemperature()
	memInfo := m.getMemoryInfo()
	swapInfo := m.getSwapInfo()
	diskInfo := m.getDiskInfo()

	return SystemStats{
		RunTime:            uptime,
		Last1:              fmt.Sprintf("%.2f", loadAvg[0]),
		Last5:              fmt.Sprintf("%.2f", loadAvg[1]),
		Last15:             fmt.Sprintf("%.2f", loadAvg[2]),
		CPUUsage:           fmt.Sprintf("%.2f", cpuUsage),
		CPUTemp:            cpuTemp,
		MemTotalSpace:      fmt.Sprintf("%.2f", float64(memInfo["total"])/1024),
		MemUsedSpace:       fmt.Sprintf("%.2f", float64(memInfo["used"])/1024),
		MemFreeSpace:       fmt.Sprintf("%.2f", float64(memInfo["total"]-memInfo["used"])/1024),
		MemUsage:           fmt.Sprintf("%.2f", float64(memInfo["used"])*100/float64(memInfo["total"])),
		SwapTotalSpace:     fmt.Sprintf("%.2f", float64(swapInfo["total"])/1024),
		SwapUsedSpace:      fmt.Sprintf("%.2f", float64(swapInfo["used"])/1024),
		SwapFreeSpace:      fmt.Sprintf("%.2f", float64(swapInfo["free"])/1024),
		DiskTotalSpace:     fmt.Sprintf("%.2f", float64(diskInfo["total"])/1024/1024),
		DiskUsedSpace:      fmt.Sprintf("%.2f", float64(diskInfo["used"])/1024/1024),
		DiskAvailableSpace: fmt.Sprintf("%.2f", float64(diskInfo["available"])/1024/1024),
		DiskUsage:          fmt.Sprintf("%.2f", float64(diskInfo["used"])*100/float64(diskInfo["total"])),
		ReceiveSpeed:       fmt.Sprintf("%.2f", receiveSpeed),
		TransmitSpeed:      fmt.Sprintf("%.2f", transmitSpeed),
		ReceiveTotal:       fmt.Sprintf("%.2f", float64(currNetRx)/1024/1024/1024),
		TransmitTotal:      fmt.Sprintf("%.2f", float64(currNetTx)/1024/1024/1024),
		LatestTime:         time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05"),
	}
}

// getNetworkStats 获取网络统计信息
func (m *Monitor) getNetworkStats() (uint64, uint64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, m.config.Interface+":") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				rx, _ := strconv.ParseUint(fields[1], 10, 64)
				tx, _ := strconv.ParseUint(fields[9], 10, 64)
				return rx, tx
			}
		}
	}
	return 0, 0
}

// getCPUStats 获取CPU统计信息
func (m *Monitor) getCPUStats() CPUStat {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return CPUStat{}
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				user, _ := strconv.ParseUint(fields[1], 10, 64)
				nice, _ := strconv.ParseUint(fields[2], 10, 64)
				system, _ := strconv.ParseUint(fields[3], 10, 64)
				idle, _ := strconv.ParseUint(fields[4], 10, 64)
				iowait, _ := strconv.ParseUint(fields[5], 10, 64)
				irq, _ := strconv.ParseUint(fields[6], 10, 64)
				softirq, _ := strconv.ParseUint(fields[7], 10, 64)
				return CPUStat{
					User:    user,
					Nice:    nice,
					System:  system,
					Idle:    idle,
					Iowait:  iowait,
					Irq:     irq,
					Softirq: softirq,
				}
			}
		}
	}
	return CPUStat{}
}

// calculateCPUUsage 计算CPU使用率
func (m *Monitor) calculateCPUUsage(prev, curr CPUStat) float64 {
	prevTotal := prev.User + prev.Nice + prev.System + prev.Idle + prev.Iowait + prev.Irq + prev.Softirq
	currTotal := curr.User + curr.Nice + curr.System + curr.Idle + curr.Iowait + curr.Irq + curr.Softirq

	prevIdle := prev.Idle + prev.Iowait
	currIdle := curr.Idle + curr.Iowait

	totalDiff := currTotal - prevTotal
	idleDiff := currIdle - prevIdle

	if totalDiff == 0 {
		return 0
	}

	return float64(totalDiff-idleDiff) * 100.0 / float64(totalDiff)
}

// getUptime 获取系统运行时间
func (m *Monitor) getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "0天0小时0分钟"
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "0天0小时0分钟"
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "0天0小时0分钟"
	}

	days := int(uptime) / 86400
	hours := (int(uptime) % 86400) / 3600
	minutes := (int(uptime) % 3600) / 60

	return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
}

// getLoadAverage 获取系统负载
func (m *Monitor) getLoadAverage() [3]float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return [3]float64{0, 0, 0}
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return [3]float64{0, 0, 0}
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return [3]float64{load1, load5, load15}
}

// getCPUTemperature 获取CPU温度
func (m *Monitor) getCPUTemperature() string {
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return "N/A"
	}

	tempStr := strings.TrimSpace(string(data))
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		return "N/A"
	}

	return fmt.Sprintf("%.1f°C", float64(temp)/1000.0)
}

// getMemoryInfo 获取内存信息
func (m *Monitor) getMemoryInfo() map[string]uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return map[string]uint64{"total": 0, "free": 0, "used": 0}
	}

	memInfo := make(map[string]uint64)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				memInfo["total"] = val
			}
		} else if strings.HasPrefix(line, "MemFree:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				memInfo["free"] = val
			}
		} else if strings.HasPrefix(line, "Buffers:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				memInfo["buffers"] = val
			}
		} else if strings.HasPrefix(line, "Cached:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				memInfo["cached"] = val
			}
		} else if strings.HasPrefix(line, "SReclaimable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				memInfo["sreclaimable"] = val
			}
		}
	}

	// 已使用内存 = 总内存 - 空闲内存 - 缓冲区 - 缓存 - 可回收内存
	memInfo["used"] = memInfo["total"] - memInfo["free"] - memInfo["buffers"] - memInfo["cached"] - memInfo["sreclaimable"]
	return memInfo
}

// getSwapInfo 获取SWAP信息
func (m *Monitor) getSwapInfo() map[string]uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return map[string]uint64{"total": 0, "free": 0, "used": 0}
	}

	swapInfo := make(map[string]uint64)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "SwapTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				swapInfo["total"] = val
			}
		} else if strings.HasPrefix(line, "SwapFree:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				swapInfo["free"] = val
			}
		}
	}

	swapInfo["used"] = swapInfo["total"] - swapInfo["free"]
	return swapInfo
}

// getDiskInfo 获取磁盘信息
func (m *Monitor) getDiskInfo() map[string]uint64 {
	cmd := exec.Command("df")
	output, err := cmd.Output()
	if err != nil {
		return map[string]uint64{"total": 0, "used": 0, "available": 0}
	}

	lines := strings.Split(string(output), "\n")
	totalSpace := uint64(0)
	usedSpace := uint64(0)
	availableSpace := uint64(0)

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		filesystem := fields[0]

		// 参考shell脚本的过滤逻辑：排除虚拟文件系统和临时文件系统
		if strings.Contains(filesystem, "-") ||
			filesystem == "none" ||
			strings.HasPrefix(filesystem, "tmpfs") ||
			strings.HasPrefix(filesystem, "devtmpfs") ||
			strings.Contains(filesystem, "by-uuid") ||
			strings.Contains(filesystem, "chroot") ||
			strings.HasPrefix(filesystem, "udev") ||
			strings.Contains(filesystem, "docker") ||
			strings.Contains(filesystem, "storage") {
			continue
		}

		// 累加所有实际磁盘的空间信息
		total, err1 := strconv.ParseUint(fields[1], 10, 64)
		used, err2 := strconv.ParseUint(fields[2], 10, 64)
		available, err3 := strconv.ParseUint(fields[3], 10, 64)

		if err1 == nil && err2 == nil && err3 == nil {
			totalSpace += total
			usedSpace += used
			availableSpace += available
		}
	}

	return map[string]uint64{
		"total":     totalSpace,
		"used":      usedSpace,
		"available": availableSpace,
	}
}
