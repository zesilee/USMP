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
	flag.Parse()

	sim := netconfsim.NewSimulator()
	sim.SetListen(*addr, *port)

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
