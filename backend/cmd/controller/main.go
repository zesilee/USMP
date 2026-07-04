package main

import (
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
	"github.com/leezesi/usmp/backend/controllers"
	netconfclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(bizv1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:         scheme,
		LeaderElection: enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// 初始化全局 NETCONF Client Pool（仅华为交换机）
	clientPool := netconfclient.NewDefaultClientPool(netconfclient.DefaultClientFactory(5))

	// BusinessVlan 意图已收编到 Stack B 的 CRD 意图源（backend/main.go：
	// crdsource.RegisterIntentSources），Actor 路径退役（P2 组4a）。

	// 设置 BusinessSwitch Reconciler（仅支持华为交换机）
	if err = (&controllers.BusinessSwitchReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		ClientPool: clientPool,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BusinessSwitch")
		os.Exit(1)
	}

	// 设置 BusinessInterface Reconciler（仅支持华为交换机）
	if err = (&controllers.BusinessInterfaceReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		ClientPool: clientPool,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BusinessInterface")
		os.Exit(1)
	}

	// 设置 BusinessRoute Reconciler（仅支持华为交换机）
	if err = (&controllers.BusinessRouteReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		ClientPool: clientPool,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BusinessRoute")
		os.Exit(1)
	}

	// 设置 NativeDeviceConfig Reconciler（通用原生配置透传
	if err = (&controllers.NativeDeviceConfigReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		ClientPool: clientPool,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NativeDeviceConfig")
		os.Exit(1)
	}

	setupLog.Info("=============================================")
	setupLog.Info("USMP Controller - 仅支持华为交换机")
	setupLog.Info("=============================================")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
