package api

import (
	"encoding/json"
	"errors"
	"strings"

	beecontext "github.com/beego/beego/v2/server/web/context"
)

// wildcardPath 取通配尾段参数并恢复 gin `*path` 语义（含前导斜杠）。
// beego `*` 的 `:splat` 不带前导斜杠，下游 YANG 路径解析依赖 `/x/y` 形态。
func wildcardPath(ctx *beecontext.Context) string {
	p := ctx.Input.Param(":splat")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// bindJSON 对齐 gin ShouldBindJSON：直读请求体解码到 v，空体/坏 JSON 返回错误。
// 不依赖 beego 全局 CopyRequestBody 配置（函数式路由下 Input.RequestBody 为空）。
func bindJSON(ctx *beecontext.Context, v interface{}) error {
	if ctx.Request == nil || ctx.Request.Body == nil {
		return errors.New("request body is empty")
	}
	return json.NewDecoder(ctx.Request.Body).Decode(v)
}
