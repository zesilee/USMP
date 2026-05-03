//go:build e2e
// +build e2e

package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
)

var _ = Describe("BusinessVlan E2E Test", func() {
	const (
		namespace = "usmp-e2e-test"
		timeout   = time.Second * 30
		interval  = time.Second * 1
	)

	Context("创建 BusinessVlan 资源", func() {
		vlanName := "test-vlan-100"

		It("应该成功创建 VLAN 配置", func() {
			By("创建 BusinessVlan 实例")
			vlan := &bizv1.BusinessVlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vlanName,
					Namespace: namespace,
				},
				Spec: bizv1.BusinessVlanSpec{
					VlanID:          100,
					DeviceID:        "switch-demo-01",
					Name:            "Test-VLAN-100",
					Description:     "E2E Test VLAN",
					Type:            bizv1.VlanTypeCommon,
					AdminStatus:     bizv1.AdminStatusUp,
					MacLearningEnabled: true,
					StatisticEnabled:   true,
				},
			}
			createObject(vlan)

			By("验证 VLAN 创建成功")
			fetched := &bizv1.BusinessVlan{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: vlanName, Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("验证 Spec 字段")
			Expect(fetched.Spec.VlanID).To(Equal(uint16(100)))
			Expect(fetched.Spec.DeviceID).To(Equal("switch-demo-01"))
			Expect(fetched.Spec.Name).To(Equal("Test-VLAN-100"))
			Expect(fetched.Spec.AdminStatus).To(Equal(bizv1.AdminStatusUp))
		})

		It("应该正确更新 VLAN 状态", func() {
			By("等待状态更新")
			vlan := &bizv1.BusinessVlan{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: vlanName, Namespace: namespace}, vlan)
				if err != nil {
					return false
				}
				return vlan.Status.Phase != ""
			}, timeout, interval).Should(BeTrue())

			By("验证 Phase 状态")
			Expect(vlan.Status.Phase).To(Or(
				Equal(bizv1.SyncPhasePending),
				Equal(bizv1.SyncPhaseSyncing),
				Equal(bizv1.SyncPhaseSynced),
			))
		})

		It("应该支持 VLAN 配置更新", func() {
			By("更新 VLAN 描述")
			vlan := &bizv1.BusinessVlan{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: vlanName, Namespace: namespace}, vlan)).Should(Succeed())

			vlan.Spec.Description = "Updated VLAN description"
			vlan.Spec.BroadcastDiscardEnabled = true
			Expect(k8sClient.Update(ctx, vlan)).Should(Succeed())

			By("验证更新生效")
			Eventually(func() string {
				updated := &bizv1.BusinessVlan{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: vlanName, Namespace: namespace}, updated)
				if err != nil {
					return ""
				}
				return updated.Spec.Description
			}, timeout, interval).Should(Equal("Updated VLAN description"))
		})
	})

	Context("批量 VLAN 创建", func() {
		It("应该支持同时创建多个 VLAN", func() {
			By("创建多个 VLAN 实例")
			for i := 200; i < 205; i++ {
				vlan := &bizv1.BusinessVlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("batch-vlan-%d", i),
						Namespace: namespace,
					},
					Spec: bizv1.BusinessVlanSpec{
						VlanID:      uint16(i),
						DeviceID:    "switch-demo-01",
						AdminStatus: bizv1.AdminStatusUp,
					},
				}
				Expect(k8sClient.Create(ctx, vlan)).Should(Succeed())
			}

			By("验证所有 VLAN 创建成功")
			list := &bizv1.BusinessVlanList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, list, client.InNamespace(namespace))
				if err != nil {
					return 0
				}
				count := 0
				for _, item := range list.Items {
					if len(item.Name) >= 10 && item.Name[:10] == "batch-vlan-" {
						count++
					}
				}
				return count
			}, timeout, interval).Should(Equal(5))
		})
	})

	Context("VLAN 类型测试", func() {
		It("应该支持 Super VLAN 类型", func() {
			By("创建 Super VLAN")
			vlan := &bizv1.BusinessVlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "super-vlan-1000",
					Namespace: namespace,
				},
				Spec: bizv1.BusinessVlanSpec{
					VlanID:      1000,
					DeviceID:    "switch-demo-01",
					Type:        bizv1.VlanTypeSuper,
					AdminStatus: bizv1.AdminStatusUp,
				},
			}
			createObject(vlan)

			By("验证 Super VLAN 创建成功")
			fetched := &bizv1.BusinessVlan{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "super-vlan-1000", Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Spec.Type).To(Equal(bizv1.VlanTypeSuper))
		})

		It("应该支持 Sub VLAN 类型", func() {
			By("创建 Sub VLAN")
			vlan := &bizv1.BusinessVlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sub-vlan-1001",
					Namespace: namespace,
				},
				Spec: bizv1.BusinessVlanSpec{
					VlanID:      1001,
					DeviceID:    "switch-demo-01",
					Type:        bizv1.VlanTypeSub,
					AdminStatus: bizv1.AdminStatusUp,
				},
			}
			createObject(vlan)

			By("验证 Sub VLAN 创建成功")
			fetched := &bizv1.BusinessVlan{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "sub-vlan-1001", Namespace: namespace}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Spec.Type).To(Equal(bizv1.VlanTypeSub))
		})
	})

	Context("删除 BusinessVlan 资源", func() {
		It("应该成功删除 VLAN 配置", func() {
			By("删除 VLAN 实例")
			vlan := &bizv1.BusinessVlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vlan-100",
					Namespace: namespace,
				},
			}
			deleteObject(vlan)

			By("验证 VLAN 已删除")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-vlan-100", Namespace: namespace}, &bizv1.BusinessVlan{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
