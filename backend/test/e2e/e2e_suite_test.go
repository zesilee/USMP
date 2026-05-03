//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "USMP E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("启动测试环境")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// 注册 Schema
	err = bizv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// 创建客户端
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// 启动管理器
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// 等待管理器就绪
	time.Sleep(2 * time.Second)

	By("创建测试命名空间")
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "usmp-e2e-test",
		},
	}
	Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("清理测试命名空间")
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "usmp-e2e-test",
		},
	}
	_ = k8sClient.Delete(ctx, ns)

	cancel()
	By("停止测试环境")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// waitForCondition 等待资源达到指定状态
func waitForCondition(obj client.Object, conditionType string, timeout time.Duration) {
	Eventually(func() bool {
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if err != nil {
			return false
		}

		// 检查状态 Phase
		if status, ok := getStatusPhase(obj); ok {
			return status == conditionType
		}
		return false
	}, timeout, time.Second).Should(BeTrue())
}

// getStatusPhase 获取资源的状态 Phase 字段
func getStatusPhase(obj client.Object) (string, bool) {
	switch o := obj.(type) {
	case *bizv1.BusinessSwitch:
		return string(o.Status.Phase), true
	case *bizv1.BusinessVlan:
		return string(o.Status.Phase), true
	case *bizv1.BusinessInterface:
		return string(o.Status.Phase), true
	case *bizv1.BusinessRoute:
		return string(o.Status.Phase), true
	case *bizv1.NativeDeviceConfig:
		return string(o.Status.Phase), true
	}
	return "", false
}

// createObject 创建测试对象
func createObject(obj client.Object) {
	Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
}

// deleteObject 删除测试对象
func deleteObject(obj client.Object) {
	Expect(k8sClient.Delete(ctx, obj)).Should(Succeed())
}
