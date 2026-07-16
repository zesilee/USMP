package intent

import (
	"os"

	"k8s.io/client-go/rest"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/leader"
)

// leaderElectionEnabled reads the BIO-08 seam switch（默认关：单副本部署零行为
// 变化；多副本上生产前置 1 开启，仅 leader 产生意图事件 → 展开/2PC/清理单点执行）。
func leaderElectionEnabled() bool {
	return os.Getenv("USMP_INTENT_LEADER_ELECTION") == "1"
}

// gateSources wraps the intent event sources behind leader election when the
// seam is enabled; disabled (default) or no cluster config passes through
// untouched。选主机制经 pkg/yang-runtime/leader 泛化实现（YR-08，本地
// leaderGatedSource 副本已删除）——本面保持独立 Lease usmp-business-intent
// 与开关不变（与原生面 usmp-native-controllers 互不干扰）。
func gateSources(cfg *rest.Config, inner controller.Source) controller.Source {
	if !leaderElectionEnabled() {
		return inner
	}
	return leader.NewGate(cfg, leader.Options{
		LeaseName: "usmp-business-intent",
		Namespace: Namespace(),
		LogPrefix: "intent",
	}).Wrap(inner)
}
