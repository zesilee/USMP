package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard API response format
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Success bool        `json:"success"`
}

// Success responds with success
func Success(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
		Success: true,
	})
}

// Error responds with error
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Success: false,
	})
}

// ErrorWithData responds with error plus a structured data payload (同信封，
// 供前端渲染细节——如归属硬锁 409 携认领意图列表)。
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    data,
		Success: false,
	})
}

// DeviceOfflineError responds with specific device offline error
func DeviceOfflineError(c *gin.Context, ip string) {
	Error(c, 503, "Device "+ip+" is offline, please check connection")
}
