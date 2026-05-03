//go:build e2e
// +build e2e

package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	usmpcorev1 "github.com/leezesi/usmp/backend/api/core/v1"
)

var _ = Describe("NativeDeviceConfig E2E Test", func() {
	const (
		namespace = "usmp-e2e-test"
		timeout   = time.Second * 30
		interval  = time.Second * 1
	)

	Context("创建 NativeDeviceConfig 资源", func() {
		configName := "test-cli-banner"

		It("应该成功创建原生设备配置", func() {
			By("创建 NativeDeviceConfig 实例")
			config := &usmpcorev1.NativeDeviceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configName,
					Namespace: namespace,
				},
				Spec: usmpcorev1.NativeDeviceConfigSpec{
					DeviceID: "192.168.1.100:830",
					Module:   "huawei-ifm",
					Config: map[string]interface{}{
						"banner": "Welcome to USMP Managed Device",
					},
				},
			}
			createObject(config)

			By("验证资源创建成功")
			fetched := &usmpcorev1.NativeDeviceConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: configName, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("验证 Spec 字段正确")
			Expect(fetched.Spec.DeviceID).To(Equal("192.168.1.100:830"))
			Expect(fetched.Spec.Module).To(Equal("huawei-ifm"))
			Expect(fetched.Spec.Config["banner"]).To(Equal("Welcome to USMP Managed Device"))
		})

		It("应该正确更新配置状态", func() {
			By("等待状态更新")
			config := &usmpcorev1.NativeDeviceConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: configName, Namespace: namespace}, config)
				if err != nil {
					return false
				}
				return config.Status.Phase != ""
			}, timeout, interval).Should(BeTrue())

			By("验证 Phase 状态设置")
			Expect(config.Status.Phase).To(Or(
				Equal(usmpcorev1.PhasePending),
				Equal(usmpcorev1.PhaseUpdating),
				Equal(usmpcorev1.PhaseReady),
				Equal(usmpcorev1.PhaseFailed),
			))
		})

		It("应该支持更新配置", func() {
			By("更新配置内容")
			config := &usmpcorev1.NativeDeviceConfig{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: configName, Namespace: namespace}, config)).Should(Succeed())

			config.Spec.Module = "huawei-vlan"
			config.Spec.Config = map[string]interface{}{
				"vlan": "100",
				"name": "VLAN-100-Updated",
			}
			Expect(k8sClient.Update(ctx, config)).Should(Succeed())

			By("验证更新生效")
			Eventually(func() string {
				updated := &usmpcorev1.NativeDeviceConfig{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: configName, Namespace: namespace}, updated)
				if err != nil {
					return ""
				}
				return updated.Spec.Module
			}, timeout, interval).Should(Equal("huawei-vlan"))
		})

		It("应该支持删除配置", func() {
			By("删除配置资源")
			config := &usmpcorev1.NativeDeviceConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configName,
					Namespace: namespace,
				},
			}
			deleteObject(config)

			By("验证资源已删除")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: configName, Namespace: namespace}, &usmpcorev1.NativeDeviceConfig{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
