package api

import (
	"net/http"

	beecontext "github.com/beego/beego/v2/server/web/context"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// RPCHandler serves rpc execution (RPC-03).
type RPCHandler struct {
	manager manager.Manager
}

// NewRPCHandler creates a new RPCHandler.
func NewRPCHandler(mgr manager.Manager) *RPCHandler {
	return &RPCHandler{manager: mgr}
}

type rpcExecuteRequest struct {
	Inputs map[string]string `json:"inputs"`
}

// Execute runs a module rpc on a device (RPC-03).
//
// It validates mandatory inputs against the rpc schema, resolves the module
// namespace, and dispatches via the client's ExecuteRPC. It does NOT write the
// config cache or trigger reconcile — an rpc is a one-shot operation with no
// desired/actual state (§8 / design D4).
//
// @Summary  执行模块 rpc（运维操作）
// @Tags     rpc
// @Accept   json
// @Produce  json
// @Param    ip     path string true "设备 IP"
// @Param    module path string true "模块（根容器名）"
// @Param    rpc    path string true "rpc 名"
// @Success  200 {object} Response "执行结果（ok/reply）"
// @Router   /rpc/{ip}/{module}/{rpc} [post]
func (h *RPCHandler) Execute(c *beecontext.Context) {
	ip := c.Input.Param(":ip")
	module := c.Input.Param(":module")
	rpcName := c.Input.Param(":rpc")

	var req rpcExecuteRequest
	_ = bindJSON(c, &req) // 缺 body → 空 inputs，由下方 mandatory 校验兜底

	// 查 rpc 定义（本地 slice，元素可取址）。
	var def *yangschema.RPCDef
	defs := yangschema.ModuleRPCs[module]
	for i := range defs {
		if defs[i].Name == rpcName {
			def = &defs[i]
			break
		}
	}
	if def == nil {
		Error(c, http.StatusNotFound, "unknown rpc: "+module+"/"+rpcName)
		return
	}

	// 校验 mandatory + 组装 inputs（保持 schema 顺序，确定 payload）。
	var inputs []client.RPCInput
	for _, in := range def.Input {
		v, ok := req.Inputs[in.Name]
		if in.Mandatory && (!ok || v == "") {
			Error(c, http.StatusBadRequest, "missing mandatory input: "+in.Name)
			return
		}
		if ok && v != "" {
			inputs = append(inputs, client.RPCInput{Name: in.Name, Value: v})
		}
	}

	// 模块命名空间（rpc payload 用；运行期 ygot schema 不含，取自构建期生成物）。
	ns := yangschema.ModuleRPCNamespace[module]

	// 解析连接 + 取 client（不写缓存、不触发对账）。
	info, _ := device.ResolveConn(h.manager.GetDeviceStore(), ip)
	cli, err := h.manager.GetClientPool().Get(info)
	if err != nil {
		Error(c, http.StatusBadGateway, "connect device: "+err.Error())
		return
	}

	res, err := cli.ExecuteRPC(c.Request.Context(), ns, rpcName, inputs)
	if err != nil {
		ErrorWithData(c, http.StatusBadGateway, "rpc failed: "+err.Error(),
			map[string]interface{}{"reply": rpcReplyString(res)})
		return
	}
	Success(c, map[string]interface{}{
		"ok":       res.OK,
		"reply":    rpcReplyString(res),
		"highRisk": def.HighRisk,
	}, "rpc executed")
}

func rpcReplyString(res *client.RPCResult) string {
	if res == nil {
		return ""
	}
	return string(res.Reply)
}
