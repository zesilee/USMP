package api

import (
	beecontext "github.com/beego/beego/v2/server/web/context"
)

// Response is the standard API response format
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Success bool        `json:"success"`
}

// Success responds with success
func Success(c *beecontext.Context, data interface{}, message string) {
	_ = c.Output.JSON(Response{
		Code:    0,
		Message: message,
		Data:    data,
		Success: true,
	}, false, false)
}

// Error responds with error
func Error(c *beecontext.Context, code int, message string) {
	_ = c.Output.JSON(Response{
		Code:    code,
		Message: message,
		Success: false,
	}, false, false)
}

// ErrorWithData responds with error plus a structured data payload (同信封，
// 供前端渲染细节——如归属硬锁 409 携认领意图列表)。
func ErrorWithData(c *beecontext.Context, code int, message string, data interface{}) {
	_ = c.Output.JSON(Response{
		Code:    code,
		Message: message,
		Data:    data,
		Success: false,
	}, false, false)
}

// DeviceOfflineError responds with specific device offline error
func DeviceOfflineError(c *beecontext.Context, ip string) {
	Error(c, 503, "Device "+ip+" is offline, please check connection")
}
