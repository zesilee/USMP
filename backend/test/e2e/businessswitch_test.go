//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
)

var _ = Describe("BusinessSwitch E2E Test", func() {
	const (
		namespace    = "usmp-e2e-test"
		switchName   = "test-switch-01"
		timeout      = time.Second * 30
		interval     = time.Second * 1
	)

	Context("创建 BusinessSwitch 资源", func() {
		It("应该成功创建并进入 Pending 状态", func() {
			By("创建 BusinessSwitch 实例")
			bs := &bizv1.BusinessSwitch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      switchName,
					Namespace: namespace,
				},
				Spec: bizv1.BusinessSwitchSpec{
					DeviceIP:   "192.168.1.100",
					Vendor:     bizv1.VendorHuawei,
					Model:      "CE6857",
					Port:       830,
					Enabled:    true,
					Owner:      "test-user",
					Location:   "Test-Rack-01",
					SyncInterval: 5,
				},
			}
			createObject(bs)

			By("验证资源创建成功")
			fetched := &bizv1.BusinessSwitch{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName(switchName, namespace), fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("验证 Spec 字段正确")
			Expect(fetched.Spec.DeviceIP).To(Equal("192.168.1.100"))
			Expect(fetched.Spec.Vendor).To(Equal(bizv1.VendorHuawei))
			Expect(fetched.Spec.Model).To(Equal("CE6857"))
			Expect(fetched.Spec.Port).To(Equal(830))
			Expect(fetched.Spec.Enabled).To(BeTrue())
		})

		It("应该正确更新 Status 字段", func() {
			By("等待状态更新")
			bs := &bizv1.BusinessSwitch{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName(switchName, namespace), bs)
				if err != nil {
					return false
				}
				return bs.Status.Phase != ""
			}, timeout, interval).Should(BeTrue())

			By("验证 Phase 状态设置")
			Expect(bs.Status.Phase).To(Or(
				Equal(bizv1.SyncPhasePending),
				Equal(bizv1.SyncPhaseSyncing),
				Equal(bizv1.SyncPhaseOffline),
			))
		})

		It("应该支持更新 Spec 字段", func() {
			By("更新设备描述")
			bs := &bizv1.BusinessSwitch{}
			Expect(k8sClient.Get(ctx, NamespacedName(switchName, namespace), bs)).Should(Succeed())

			bs.Spec.Description = "Updated description"
			bs.Spec.Owner = "new-owner"
			Expect(k8sClient.Update(ctx, bs)).Should(Succeed())

			By("验证更新生效")
			Eventually(func() string {
				updated := &bizv1.BusinessSwitch{}
				err := k8sClient.Get(ctx, NamespacedName(switchName, namespace), updated)
				if err != nil {
					return ""
				}
				return updated.Spec.Description
			}, timeout, interval).Should(Equal("Updated description"))
		})
	})

	Context("删除 BusinessSwitch 资源", func() {
		It("应该成功删除资源", func() {
			By("删除 BusinessSwitch 实例")
			bs := &bizv1.BusinessSwitch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      switchName,
					Namespace: namespace,
				},
			}
			deleteObject(bs)

			By("验证资源已删除")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName(switchName, namespace), &bizv1.BusinessSwitch{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("多设备管理", func() {
		It("应该支持同时管理多个交换机", func() {
			By("创建多个 BusinessSwitch 实例")
			for i := 1; i <= 3; i++ {
				bs := &bizv1.BusinessSwitch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-switch-%d",
						Namespace: namespace,
					},
					Spec: bizv1.BusinessSwitchSpec{
						DeviceIP:   fmt.Sprintf("192.168.1.%d", 10+i),
						Vendor:     bizv1.VendorHuawei,
						Enabled:    true,
					},
				}
				createObject(bs)
			}

			By("验证所有实例创建成功")
			list := &bizv1.BusinessSwitchList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, list, client.InNamespace(namespace))
				if err != nil {
					return 0
				}
				count := 0
				for _, item := range list.Items {
					if len(item.Name) >= 14 && item.Name[:14] == "multi-switch-" {
						count++
					}
				}
				return count
			}, timeout, interval).Should(Equal(3))
		})
	})
})

// NamespacedName 辅助函数
func NamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
}
