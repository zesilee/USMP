// Command netconf-simulator runs the NETCONF simulated network element as a
// standalone, deployable process (aligns with backend/deploy/manifests/netconf-simulator).
//
// The simulator core (backend/simulator/netconfsim) carries no testing/testify
// dependency, so this binary links cleanly. Test-only assertion helpers live in
// backend/simulator/netconfsim/testsupport.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/leezesi/usmp/backend/simulator/netconfsim"
)

func main() {
	addr := flag.String("addr", "0.0.0.0", "bind address")
	port := flag.Int("port", 830, "NETCONF SSH port (0 = random free port)")
	seed := flag.Bool("seed", true, "load the demo IFM running-config seed (5 interfaces)")
	flag.Parse()

	sim := netconfsim.NewSimulator()
	sim.SetListen(*addr, *port)
	if *seed {
		// staging 演示种子：3 main-interface/200GE/up + 2 sub-interface/Vlanif/down，
		// 供通用模块控制台的表格/高级搜索/行级 when 冒烟断言。
		sim.SetRunningConfigXML([]byte(netconfsim.DemoSeedConfig))
	}

	if err := sim.Start(); err != nil {
		log.Fatalf("start netconf-simulator: %v", err)
	}
	defer sim.Stop()

	log.Printf("netconf-simulator listening on %s:%d (user=%s pass=%s)",
		sim.Addr(), sim.Port(), sim.Username(), sim.Password())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Println("netconf-simulator shutting down")
}
