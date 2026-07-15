package netconfsim

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// NS-07 — confirmed-commit 仿真：带 <confirmed/> 的 commit 将 candidate 提升为
// running 并启动确认计时器；超时未确认回滚到提交前快照；确认 commit 转正。
// hello 按 ScenarioConfig 开关宣告 :confirmed-commit capability。

func ccServer() *sshServer {
	return &sshServer{store: newTreeDatastore(), scenario: NewScenarioConfig()}
}

func rpcMsg(id, body string) string {
	return fmt.Sprintf(`<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="%s">%s</rpc>`, id, body)
}

func editCandidate(id, payload string) string {
	return rpcMsg(id, `<edit-config><target><candidate/></target><config>`+payload+`</config></edit-config>`)
}

func commitConfirmed(id string, timeoutSec int) string {
	return rpcMsg(id, fmt.Sprintf(`<commit><confirmed/><confirm-timeout>%d</confirm-timeout></commit>`, timeoutSec))
}

func plainCommit(id string) string {
	return rpcMsg(id, `<commit/>`)
}

func mustOK(t *testing.T, resp, ctx string) {
	t.Helper()
	if !strings.Contains(resp, "<ok/>") {
		t.Fatalf("%s: want <ok/>, got %s", ctx, resp)
	}
}

// waitRunning polls GetRunning until cond(running) or timeout.
func waitRunning(t *testing.T, s *sshServer, timeout time.Duration, cond func(string) bool) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		last = string(s.store.GetRunning())
		if cond(last) {
			return last
		}
		time.Sleep(20 * time.Millisecond)
	}
	return last
}

// hello 宣告 :confirmed-commit capability（开关可关）。
func TestHelloConfirmedCommitCapability(t *testing.T) {
	caps := func(h *Hello) map[string]bool {
		got := make(map[string]bool)
		for _, c := range h.Capabilities.Capabilities {
			got[c.URN] = true
		}
		return got
	}
	const urn = "urn:ietf:params:netconf:capability:confirmed-commit:1.1"
	if !caps(buildHello(1, nil, true))[urn] {
		t.Errorf("hello should advertise %s when enabled", urn)
	}
	if caps(buildHello(1, nil, false))[urn] {
		t.Errorf("hello should NOT advertise %s when disabled", urn)
	}
}

// 场景 1：确认转正——confirmed-commit 后超时内确认 commit，配置保持不回滚。
func TestConfirmedCommitThenConfirmKeepsConfig(t *testing.T) {
	s := ccServer()
	mustOK(t, s.handleRequest(editCandidate("1", `<top><item>alpha</item></top>`)), "edit")
	mustOK(t, s.handleRequest(commitConfirmed("2", 1)), "confirmed commit")

	if run := string(s.store.GetRunning()); !strings.Contains(run, "alpha") {
		t.Fatalf("running should contain committed config before confirm, got %s", run)
	}
	mustOK(t, s.handleRequest(plainCommit("3")), "confirming commit")

	// 原确认窗口过后仍不回滚。
	time.Sleep(1300 * time.Millisecond)
	if run := string(s.store.GetRunning()); !strings.Contains(run, "alpha") {
		t.Fatalf("running rolled back despite confirming commit, got %s", run)
	}
}

// 场景 2：超时未确认——running 回滚到提交前快照，candidate 一并复位。
func TestConfirmedCommitTimeoutRollsBack(t *testing.T) {
	s := ccServer()
	// 预置初始 running。
	mustOK(t, s.handleRequest(editCandidate("1", `<top><item>base</item></top>`)), "seed edit")
	mustOK(t, s.handleRequest(plainCommit("2")), "seed commit")

	mustOK(t, s.handleRequest(editCandidate("3", `<top><risky>beta</risky></top>`)), "edit")
	mustOK(t, s.handleRequest(commitConfirmed("4", 1)), "confirmed commit")
	if run := string(s.store.GetRunning()); !strings.Contains(run, "beta") {
		t.Fatalf("running should contain config during confirm window, got %s", run)
	}

	run := waitRunning(t, s, 3*time.Second, func(r string) bool { return !strings.Contains(r, "beta") })
	if strings.Contains(run, "beta") {
		t.Fatalf("running should roll back after confirm timeout, got %s", run)
	}
	if !strings.Contains(run, "base") {
		t.Fatalf("running should return to pre-commit snapshot, got %s", run)
	}
	if cand := string(s.store.GetCandidate()); strings.Contains(cand, "beta") {
		t.Fatalf("candidate should reset to snapshot on rollback, got %s", cand)
	}
}

// 场景 2b（RFC 6241 §8.4）：后续 confirmed-commit 延长计时但保留最初快照——
// 最终超时回滚到链条起点，而非中间态。
func TestConfirmedCommitExtendKeepsOriginalSnapshot(t *testing.T) {
	s := ccServer()
	mustOK(t, s.handleRequest(editCandidate("1", `<top><item>base</item></top>`)), "seed edit")
	mustOK(t, s.handleRequest(plainCommit("2")), "seed commit")

	mustOK(t, s.handleRequest(editCandidate("3", `<top><step>one</step></top>`)), "edit one")
	mustOK(t, s.handleRequest(commitConfirmed("4", 1)), "confirmed commit one")
	mustOK(t, s.handleRequest(editCandidate("5", `<top><step2>two</step2></top>`)), "edit two")
	mustOK(t, s.handleRequest(commitConfirmed("6", 1)), "confirmed commit two")

	run := waitRunning(t, s, 3*time.Second, func(r string) bool { return !strings.Contains(r, "two") })
	if strings.Contains(run, "one") || strings.Contains(run, "two") {
		t.Fatalf("rollback should return to original snapshot, got %s", run)
	}
	if !strings.Contains(run, "base") {
		t.Fatalf("rollback lost original snapshot, got %s", run)
	}
}

// 场景 3：能力开关——关闭后 hello 不宣告（上面已断言），且 confirmed-commit RPC 明确报错。
func TestConfirmedCommitDisabledRejectsRPC(t *testing.T) {
	s := ccServer()
	s.scenario.DisableConfirmedCommit = true
	mustOK(t, s.handleRequest(editCandidate("1", `<top><item>x</item></top>`)), "edit")
	resp := s.handleRequest(commitConfirmed("2", 1))
	if !strings.Contains(resp, "<rpc-error>") {
		t.Fatalf("confirmed commit should be rejected when disabled, got %s", resp)
	}
	// 普通 commit 不受影响。
	mustOK(t, s.handleRequest(plainCommit("3")), "plain commit")
}

// 无待确认事务时的普通 commit 行为不变（回归防线）。
func TestPlainCommitWithoutPendingConfirmStillCommits(t *testing.T) {
	s := ccServer()
	mustOK(t, s.handleRequest(editCandidate("1", `<top><item>plain</item></top>`)), "edit")
	mustOK(t, s.handleRequest(plainCommit("2")), "commit")
	if run := string(s.store.GetRunning()); !strings.Contains(run, "plain") {
		t.Fatalf("plain commit should promote candidate, got %s", run)
	}
}

// 并发防线（R09）：确认计时器触发回滚的同时并发读写不产生竞态（-race 兜底）。
func TestConfirmedCommitConcurrentAccess(t *testing.T) {
	s := ccServer()
	mustOK(t, s.handleRequest(editCandidate("1", `<top><item>c</item></top>`)), "edit")
	mustOK(t, s.handleRequest(commitConfirmed("2", 1)), "confirmed commit")

	var wg sync.WaitGroup
	stop := make(chan struct{})
	time.AfterFunc(1500*time.Millisecond, func() { close(stop) })
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = s.store.GetRunning()
					_ = s.handleRequest(rpcMsg("9", `<get-config><source><running/></source></get-config>`))
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}
	wg.Wait()
}
