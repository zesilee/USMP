package main

import (
	"flag"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	corev1 "github.com/leezesi/usmp/backend/api/core/v1"
	bizv1 "github.com/leezesi/usmp/backend/api/biz/v1"
	"github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/actor"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = corev1.AddToScheme(scheme)
	_ = bizv1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "usmp-controller-lock",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize device client pool
	clientPool := client.NewDefaultClientPool(client.DefaultClientFactory(30 * time.Second))

	// Initialize simple in-memory config store
	configStore := NewInMemoryConfigStore()

	// Initialize Actor Manager
	actorManager := actor.NewActorManager(clientPool, configStore)
	setupLog.Info("actor manager initialized")

	// Setup VLAN reconciler (Actor-based)
	if err := vlan.NewActorBasedVlanReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		actorManager,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BusinessVlan")
		os.Exit(1)
	}
	setupLog.Info("VLAN controller setup completed")

	// Health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// InMemoryConfigStore is a simple in-memory implementation of reconcile.ConfigStore
type InMemoryConfigStore struct {
	data map[string]map[string]interface{}
}

func NewInMemoryConfigStore() *InMemoryConfigStore {
	return &InMemoryConfigStore{
		data: make(map[string]map[string]interface{}),
	}
}

func (s *InMemoryConfigStore) Get(deviceID, path string) (interface{}, error) {
	if deviceData, ok := s.data[deviceID]; ok {
		if value, ok := deviceData[path]; ok {
			return value, nil
		}
	}
	return nil, nil // Return nil for non-existent config
}

func (s *InMemoryConfigStore) Set(deviceID, path string, value interface{}) error {
	if _, ok := s.data[deviceID]; !ok {
		s.data[deviceID] = make(map[string]interface{})
	}
	s.data[deviceID][path] = value
	return nil
}

func (s *InMemoryConfigStore) Delete(deviceID, path string) error {
	if deviceData, ok := s.data[deviceID]; ok {
		delete(deviceData, path)
	}
	return nil
}

func (s *InMemoryConfigStore) List(deviceID string) ([]string, error) {
	if deviceData, ok := s.data[deviceID]; ok {
		paths := make([]string, 0, len(deviceData))
		for path := range deviceData {
			paths = append(paths, path)
		}
		return paths, nil
	}
	return []string{}, nil
}

func (s *InMemoryConfigStore) ListDevices() ([]string, error) {
	devices := make([]string, 0, len(s.data))
	for deviceID := range s.data {
		devices = append(devices, deviceID)
	}
	return devices, nil
}
