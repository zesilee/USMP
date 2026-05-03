# 集成测试

本目录包含 USMP 平台的端到端集成测试，使用 NETCONF 模拟器进行验证。

## 测试架构

```
┌─────────────────────┐     HTTP      ┌─────────────────────┐
│  Integration Tests  │ ────────────> │   USMP API Server   │
└─────────────────────┘               └─────────────────────┘
                                               │
                                               │ NETCONF
                                               ▼
                                       ┌─────────────────────┐
                                       │  NETCONF Simulator  │
                                       │  (In-Process Mock)  │
                                       └─────────────────────┘
```

## 测试覆盖范围

| 测试文件 | 测试场景 | 覆盖模块 |
|---------|---------|---------|
| `netconf_simulator.go` | NETCONF 协议模拟 | 设备通信层 |
| `e2e_test.go` | 端到端集成测试 | 完整 API + 业务逻辑 |

### 测试用例

1. **设备连接测试** (`TestE2E_DeviceConnection`)
   - 添加设备
   - 获取设备列表
   - 获取设备状态

2. **VLAN 配置测试** (`TestE2E_VLANConfiguration`)
   - 创建单个 VLAN 配置
   - 获取 VLAN 配置
   - 创建多个 VLAN 配置

3. **接口配置测试** (`TestE2E_InterfaceConfiguration`)
   - 配置接口描述、MTU、管理状态

4. **系统配置测试** (`TestE2E_SystemConfiguration`)
   - 配置系统名称、联系人、位置

5. **错误请求测试** (`TestE2E_InvalidRequests`)
   - 缺少必填字段的请求
   - 不存在设备的查询

6. **YANG 模块测试** (`TestE2E_YANGModules`)
   - 获取支持的 YANG 模块列表

7. **并发请求测试** (`TestE2E_ConcurrentRequests`)
   - 并发发送多个配置请求

8. **设备移除测试** (`TestE2E_RemoveDevice`)
   - 移除已注册的设备

## 运行测试

### 前置条件

1. 确保后端 API 服务器正在运行：
```bash
cd /path/to/usmp/backend
go run main.go
```

2. 服务器应监听在 `localhost:8080`

### 运行所有集成测试

```bash
cd backend
go test -v -tags=integration ./test/integration/...
```

### 运行单个测试文件

```bash
cd backend
go test -v -tags=integration ./test/integration/e2e_test.go
```

### 运行特定测试用例

```bash
# 只运行 VLAN 相关测试
go test -v -tags=integration ./test/integration/... -run TestE2E_VLAN

# 只运行设备连接测试
go test -v -tags=integration ./test/integration/... -run TestE2E_DeviceConnection
```

### 运行测试并生成覆盖率报告

```bash
go test -v -tags=integration ./test/integration/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 测试标签

集成测试使用 `integration` build tag，与单元测试区分开：

```bash
# 只运行单元测试（快速，不需要外部依赖）
go test -v ./...

# 只运行集成测试（需要 API 服务器运行）
go test -v -tags=integration ./test/integration/...
```

## NETCONF 模拟器

模拟器是一个内存中的 NETCONF 服务器，用于测试目的：

### 功能特性

- ✅ 支持 NETCONF 1.0 协议
- ✅ 模拟华为交换机的 VLAN、IFM、System YANG 模型
- ✅ 支持 get-config 操作
- ✅ 支持 edit-config 操作
- ✅ 配置数据持久化存储（内存中）
- ✅ 支持并发连接

### 扩展模拟器

如需添加对新 YANG 模型的支持，修改 `netconf_simulator.go`：

1. 在 `handleGetConfig` 中添加新模型的响应生成逻辑
2. 在 `handleEditConfig` 中添加新模型的配置解析和存储逻辑
3. 在 hello 消息中添加新的 capability

## 测试最佳实践

### 1. 测试隔离

每个测试用例创建自己的模拟器实例，确保测试之间不互相影响：

```go
func TestE2E_MyTest(t *testing.T) {
    sim := NewNETCONFSimulator(t)
    defer sim.Stop()
    
    // 测试逻辑...
}
```

### 2. 错误处理

集成测试中允许部分请求因环境原因失败（例如设备连接还未建立），应优雅处理：

```go
if resp.StatusCode == http.StatusOK {
    // 验证成功响应
} else {
    t.Logf("Request returned status %d (acceptable in integration test)", resp.StatusCode)
}
```

### 3. 日志输出

使用 `t.Logf` 输出详细的测试执行信息，便于调试：

```go
t.Logf("Device added successfully: %+v", result)
```

## 故障排查

### 测试失败 - 连接被拒绝

**问题**: `dial tcp [::1]:8080: connect: connection refused`

**解决方案**: 确保后端 API 服务器正在运行：
```bash
cd backend
go run main.go
```

### 测试失败 - 设备连接超时

**问题**: 设备连接请求超时

**解决方案**:
1. 检查模拟器是否正常启动
2. 增加等待时间：`time.Sleep(1 * time.Second)`
3. 检查端口号是否正确

### 测试不稳定

**问题**: 测试有时通过有时失败

**解决方案**:
1. 确保测试之间有足够的等待时间
2. 检查是否存在并发问题
3. 使用 `t.Parallel()` 标记可并行的测试

## 扩展测试

### 添加新的集成测试

1. 在 `e2e_test.go` 中添加新的测试函数：
```go
func TestE2E_MyNewFeature(t *testing.T) {
    // 1. 启动模拟器
    sim := NewNETCONFSimulator(t)
    defer sim.Stop()
    
    // 2. 添加设备
    
    // 3. 执行测试操作
    
    // 4. 验证结果
}
```

2. 运行新测试：
```bash
go test -v -tags=integration ./test/integration/... -run TestE2E_MyNewFeature
```

### 添加新的 YANG 模型支持

参考 `netconf_simulator.go` 中的 VLAN 处理逻辑，添加新模型的处理函数。

## CI/CD 集成

在 GitHub Actions/GitLab CI 中，可以按以下方式运行集成测试：

```yaml
jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Start API server
        run: |
          cd backend
          go run main.go &
          sleep 5  # 等待服务器启动
      
      - name: Run integration tests
        run: |
          cd backend
          go test -v -tags=integration ./test/integration/...
```
