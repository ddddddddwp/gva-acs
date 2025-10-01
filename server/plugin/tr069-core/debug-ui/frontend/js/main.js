// TR069 调试工具主要JavaScript文件

class TR069DebugTool {
    constructor() {
        this.apiBaseUrl = '/api';
        this.currentPage = 'parser';
        this.init();
    }

    init() {
        this.setupNavigation();
        this.setupEventListeners();
        this.showPage('parser');
        this.checkServerStatus();
    }

    // 设置导航
    setupNavigation() {
        const sidebarItems = document.querySelectorAll('.sidebar-item');
        sidebarItems.forEach(item => {
            item.addEventListener('click', (e) => {
                e.preventDefault();
                const page = item.getAttribute('data-page');
                if (page) {
                    this.showPage(page);
                    this.setActiveNav(item);
                }
            });
        });
    }

    // 设置事件监听器
    setupEventListeners() {
        // 消息解析
        const parseBtn = document.getElementById('parse-btn');
        if (parseBtn) {
            parseBtn.addEventListener('click', () => this.parseMessage());
        }

        // 消息构建
        const buildBtn = document.getElementById('build-btn');
        if (buildBtn) {
            buildBtn.addEventListener('click', () => this.buildMessage());
        }

        // 清空按钮
        const clearBtns = document.querySelectorAll('.clear-btn');
        clearBtns.forEach(btn => {
            btn.addEventListener('click', (e) => {
                const targetId = btn.getAttribute('data-target');
                if (targetId) {
                    document.getElementById(targetId).value = '';
                }
            });
        });

        // 示例按钮
        const exampleBtns = document.querySelectorAll('.example-btn');
        exampleBtns.forEach(btn => {
            btn.addEventListener('click', (e) => {
                const exampleType = btn.getAttribute('data-example');
                this.loadExample(exampleType);
            });
        });

        // 刷新状态按钮
        const refreshStatusBtn = document.getElementById('refresh-status-btn');
        if (refreshStatusBtn) {
            refreshStatusBtn.addEventListener('click', () => this.checkServerStatus());
        }
    }

    // 显示页面
    showPage(pageId) {
        // 隐藏所有页面
        const pages = document.querySelectorAll('.content-page');
        pages.forEach(page => page.classList.remove('active'));

        // 显示目标页面
        const targetPage = document.getElementById(`${pageId}-page`);
        if (targetPage) {
            targetPage.classList.add('active');
            this.currentPage = pageId;
            this.updateNavbarTitle(pageId);
        }
    }

    // 设置活跃导航
    setActiveNav(activeItem) {
        const sidebarItems = document.querySelectorAll('.sidebar-item');
        sidebarItems.forEach(item => item.classList.remove('active'));
        activeItem.classList.add('active');
    }

    // 更新导航栏标题
    updateNavbarTitle(pageId) {
        const titles = {
            'parser': 'TR069 消息解析',
            'builder': 'TR069 消息构建',
            'status': '库状态监控',
            'logs': '调试日志'
        };
        
        const titleElement = document.querySelector('.navbar-title');
        if (titleElement && titles[pageId]) {
            titleElement.textContent = titles[pageId];
        }
    }

    // 解析消息
    async parseMessage() {
        const messageInput = document.getElementById('xml-input');
        const parseBtn = document.getElementById('parse-btn');
        const resultArea = document.getElementById('parse-result');

        if (!messageInput || !messageInput.value.trim()) {
            this.showError(resultArea, '请输入要解析的TR069消息');
            return;
        }

        try {
            parseBtn.disabled = true;
            parseBtn.innerHTML = '<span class="loading"></span> 解析中...';
            
            const response = await fetch(`${this.apiBaseUrl}/parse`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    message: messageInput.value.trim()
                })
            });

            const result = await response.json();
            
            if (response.ok) {
                this.showParseResult(resultArea, result);
                this.addLog('info', `消息解析成功: ${result.method || 'Unknown'}`);
            } else {
                this.showError(resultArea, result.error || '解析失败');
                this.addLog('error', `消息解析失败: ${result.error}`);
            }
        } catch (error) {
            this.showError(resultArea, `网络错误: ${error.message}`);
            this.addLog('error', `网络错误: ${error.message}`);
        } finally {
            parseBtn.disabled = false;
            parseBtn.innerHTML = '<i class="bi bi-play-circle"></i> 解析消息';
        }
    }

    // 构建消息
    async buildMessage() {
        const methodInput = document.getElementById('rpc-method');
        const parametersInput = document.getElementById('build-params');
        const buildBtn = document.getElementById('build-btn');
        const resultArea = document.getElementById('build-result');

        if (!methodInput || !methodInput.value.trim()) {
            this.showError(resultArea, '请选择RPC方法');
            return;
        }

        let parameters = {};
        if (parametersInput && parametersInput.value.trim()) {
            try {
                parameters = JSON.parse(parametersInput.value.trim());
            } catch (error) {
                this.showError(resultArea, '参数格式错误，请输入有效的JSON');
                return;
            }
        }

        try {
            buildBtn.disabled = true;
            buildBtn.innerHTML = '<span class="loading"></span> 构建中...';
            
            const response = await fetch(`${this.apiBaseUrl}/build`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    method: methodInput.value.trim(),
                    parameters: parameters
                })
            });

            const result = await response.json();
            
            if (response.ok) {
                this.showBuildResult(resultArea, result);
                this.addLog('info', `消息构建成功: ${methodInput.value.trim()}`);
            } else {
                this.showError(resultArea, result.error || '构建失败');
                this.addLog('error', `消息构建失败: ${result.error}`);
            }
        } catch (error) {
            this.showError(resultArea, `网络错误: ${error.message}`);
            this.addLog('error', `网络错误: ${error.message}`);
        } finally {
            buildBtn.disabled = false;
            buildBtn.innerHTML = '<i class="bi bi-hammer"></i> 构建消息';
        }
    }

    // 检查服务器状态
    async checkServerStatus() {
        const statusIndicator = document.getElementById('server-status');
        const libraryVersion = document.getElementById('library-version');
        const uptime = document.getElementById('server-uptime');
        const refreshBtn = document.getElementById('refresh-status-btn');

        try {
            if (refreshBtn) {
                refreshBtn.disabled = true;
                refreshBtn.innerHTML = '<span class="loading"></span> 检查中...';
            }

            const response = await fetch(`${this.apiBaseUrl}/status`);
            const result = await response.json();

            if (response.ok) {
                if (statusIndicator) {
                    statusIndicator.className = 'status-indicator online';
                    statusIndicator.textContent = '在线';
                }
                if (libraryVersion) {
                    libraryVersion.textContent = result.version || 'Unknown';
                }
                if (uptime) {
                    uptime.textContent = result.uptime || 'Unknown';
                }
                this.addLog('success', '服务器状态检查成功');
            } else {
                if (statusIndicator) {
                    statusIndicator.className = 'status-indicator offline';
                    statusIndicator.textContent = '离线';
                }
                this.addLog('error', '服务器状态检查失败');
            }
        } catch (error) {
            if (statusIndicator) {
                statusIndicator.className = 'status-indicator offline';
                statusIndicator.textContent = '连接失败';
            }
            this.addLog('error', `服务器连接失败: ${error.message}`);
        } finally {
            if (refreshBtn) {
                refreshBtn.disabled = false;
                refreshBtn.innerHTML = '<i class="bi bi-arrow-clockwise"></i> 刷新状态';
            }
        }
    }

    // 显示解析结果
    showParseResult(container, result) {
        if (!container) return;

        container.innerHTML = `
            <div class="result-header">
                <h4>解析结果</h4>
                <span class="status-indicator online">成功</span>
            </div>
            <div class="result-content">
                <div class="card">
                    <div class="card-header">
                        <h5>基本信息</h5>
                    </div>
                    <div class="card-body">
                        <p><strong>RPC方法:</strong> ${result.method || 'Unknown'}</p>
                        <p><strong>消息类型:</strong> ${result.type || 'Unknown'}</p>
                        <p><strong>消息ID:</strong> ${result.id || 'N/A'}</p>
                    </div>
                </div>
                ${result.parameters ? `
                <div class="card">
                    <div class="card-header">
                        <h5>参数列表</h5>
                    </div>
                    <div class="card-body">
                        <div class="code-block json">${JSON.stringify(result.parameters, null, 2)}</div>
                    </div>
                </div>
                ` : ''}
                ${result.raw_xml ? `
                <div class="card">
                    <div class="card-header">
                        <h5>原始XML</h5>
                    </div>
                    <div class="card-body">
                        <div class="code-block xml">${this.escapeHtml(result.raw_xml)}</div>
                    </div>
                </div>
                ` : ''}
            </div>
        `;
    }

    // 显示构建结果
    showBuildResult(container, result) {
        if (!container) return;

        container.innerHTML = `
            <div class="result-header">
                <h4>构建结果</h4>
                <span class="status-indicator online">成功</span>
            </div>
            <div class="result-content">
                <div class="card">
                    <div class="card-header">
                        <h5>生成的XML消息</h5>
                    </div>
                    <div class="card-body">
                        <div class="code-block xml">${this.escapeHtml(result.xml || result.message || '')}</div>
                    </div>
                </div>
                <div class="card">
                    <div class="card-header">
                        <h5>消息统计</h5>
                    </div>
                    <div class="card-body">
                        <p><strong>消息长度:</strong> ${(result.xml || result.message || '').length} 字符</p>
                        <p><strong>构建时间:</strong> ${result.build_time || 'N/A'}</p>
                    </div>
                </div>
            </div>
        `;
    }

    // 显示错误
    showError(container, message) {
        if (!container) return;

        container.innerHTML = `
            <div class="result-header">
                <h4>错误</h4>
                <span class="status-indicator offline">失败</span>
            </div>
            <div class="error-message">
                ${this.escapeHtml(message)}
            </div>
        `;
    }

    // 加载示例
    loadExample(exampleType) {
        const examples = {
            'inform': {
                message: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
    <soap:Header>
        <cwmp:ID soap:mustUnderstand="1">12345</cwmp:ID>
    </soap:Header>
    <soap:Body>
        <cwmp:Inform>
            <DeviceId>
                <Manufacturer>ExampleManufacturer</Manufacturer>
                <OUI>123456</OUI>
                <ProductClass>ExampleProduct</ProductClass>
                <SerialNumber>SN123456789</SerialNumber>
            </DeviceId>
            <Event soap:arrayType="cwmp:EventStruct[1]">
                <EventStruct>
                    <EventCode>1 BOOT</EventCode>
                    <CommandKey></CommandKey>
                </EventStruct>
            </Event>
            <MaxEnvelopes>1</MaxEnvelopes>
            <CurrentTime>2024-01-01T00:00:00Z</CurrentTime>
            <RetryCount>0</RetryCount>
            <ParameterList soap:arrayType="cwmp:ParameterValueStruct[0]">
            </ParameterList>
        </cwmp:Inform>
    </soap:Body>
</soap:Envelope>`
            },
            'getParameterValues': {
                method: 'GetParameterValues',
                parameters: {
                    "ParameterNames": [
                        "Device.DeviceInfo.Manufacturer",
                        "Device.DeviceInfo.ModelName",
                        "Device.DeviceInfo.SoftwareVersion"
                    ]
                }
            },
            'setParameterValues': {
                method: 'SetParameterValues',
                parameters: {
                    "ParameterList": [
                        {
                            "Name": "Device.ManagementServer.URL",
                            "Value": "http://acs.example.com:7547/"
                        },
                        {
                            "Name": "Device.ManagementServer.Username",
                            "Value": "admin"
                        }
                    ]
                }
            }
        };

        if (this.currentPage === 'parser' && examples[exampleType] && examples[exampleType].message) {
            const messageInput = document.getElementById('xml-input');
            if (messageInput) {
                messageInput.value = examples[exampleType].message;
            }
        } else if (this.currentPage === 'builder' && examples[exampleType]) {
            const methodInput = document.getElementById('rpc-method');
            const parametersInput = document.getElementById('build-params');
            
            if (methodInput && examples[exampleType].method) {
                methodInput.value = examples[exampleType].method;
            }
            if (parametersInput && examples[exampleType].parameters) {
                parametersInput.value = JSON.stringify(examples[exampleType].parameters, null, 2);
            }
        }
    }

    // 添加日志
    addLog(level, message) {
        const logContainer = document.getElementById('debug-logs');
        if (!logContainer) return;

        const timestamp = new Date().toLocaleTimeString();
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry ${level}`;
        logEntry.innerHTML = `<span class="log-timestamp">[${timestamp}]</span>${this.escapeHtml(message)}`;

        logContainer.appendChild(logEntry);
        logContainer.scrollTop = logContainer.scrollHeight;

        // 限制日志条数
        const maxLogs = 100;
        while (logContainer.children.length > maxLogs) {
            logContainer.removeChild(logContainer.firstChild);
        }
    }

    // HTML转义
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // 格式化XML
    formatXml(xml) {
        const PADDING = ' '.repeat(2);
        const reg = /(>)(<)(\/*)/g;
        let formatted = xml.replace(reg, '$1\r\n$2$3');
        let pad = 0;
        
        return formatted.split('\r\n').map(line => {
            let indent = 0;
            if (line.match(/.+<\/\w[^>]*>$/)) {
                indent = 0;
            } else if (line.match(/^<\/\w/) && pad !== 0) {
                pad -= 1;
            } else if (line.match(/^<\w[^>]*[^\/]>.*$/)) {
                indent = 1;
            } else {
                indent = 0;
            }
            
            const padding = PADDING.repeat(pad);
            pad += indent;
            return padding + line;
        }).join('\r\n');
    }
}

// 初始化应用
document.addEventListener('DOMContentLoaded', () => {
    window.debugTool = new TR069DebugTool();
});

// 全局工具函数
window.clearInput = function(inputId) {
    const input = document.getElementById(inputId);
    if (input) {
        input.value = '';
    }
};

window.copyToClipboard = function(text) {
    navigator.clipboard.writeText(text).then(() => {
        // 可以添加复制成功的提示
        console.log('已复制到剪贴板');
    }).catch(err => {
        console.error('复制失败:', err);
    });
};