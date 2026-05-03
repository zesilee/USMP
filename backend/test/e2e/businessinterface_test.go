//go:build e2e
// +build e2e

package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bizv1 "github.com/leezesi/usmp/backend/api/biz/v1"
)

var _ = Describe("BusinessInterface E2E Test", func() {
	const (
		namespace = "usmp-e2e-test"
		timeout   = time.Second * 30
		interval  = time.Second * 1
	)

	Context("创建 BusinessInterface 资源", func() {
		ifName := "test-interface-g0"

		It("应该成功创建接口配置", func() {
			By("创建 BusinessInterface 实例")
			iface := &bizv1.BusinessInterface{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ifName,
					Namespace: namespace,
				},
				Spec: bizv1.BusinessInterfaceSpec{
					DeviceID:    "192.168.1.100:830",
					IfName:      "GigabitEthernet0/0/1",
					Description: "E2E Test Interface",
					AdminStatus: bizv1.InterfaceAdminStatusUp,
					Mode:        bizv1.InterfaceModeAccess,
					AccessVlan:  100,
					MTU:         1500,
				},
			}
			createObject(iface)

			By("验证资源创建成功")
			fetched := &bizv1.BusinessInterface{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName(ifName, namespace), fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("验证 Spec 字段正确")
			Expect(fetched.Spec.DeviceID).To(Equal("192.168.1.100:830"))
			Expect(fetched.Spec.IfName).To(Equal("GigabitEthernet0/0/1"))
			Expect(fetched.Spec.Description).To(Equal("E2E Test Interface"))
			Expect(fetched.Spec.AdminStatus).To(Equal(bizv1.InterfaceAdminStatusUp))
			Expect(fetched.Spec.Mode).To(Equal(bizv1.InterfaceModeAccess))
			Expect(fetched.Spec.AccessVlan).To(Equal(uint16(100)))
			Expect(fetched.Spec.MTU).To(Equal(uint32(1500)))
		})

		It("应该支持 Trunk 模式配置", func() {
			trunkIface := &bizv1.BusinessInterface{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ifName + "-trunk",
					Namespace: namespace,
				},
				Spec: bizv1.BusinessInterfaceSpec{
					DeviceID:    "192.168.1.100:830",
					IfName:      "GigabitEthernet0/0/2",
					AdminStatus: bizv1.InterfaceAdminStatusUp,
					Mode:        bizv1.InterfaceModeTrunk,
					TrunkVlans:  []uint16{100, 200, 300},
				},
			}
			createObject(trunkIface)

			By("验证 Trunk 模式配置正确")
			fetched := &bizv1.BusinessInterface{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName(ifName+"-trunk", namespace), fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(fetched.Spec.Mode).To(Equal(bizv1.InterfaceModeTrunk))
			Expect(fetched.Spec.TrunkVlans).To(ContainElements(uint16(100), uint16(200), uint16(300)))
		})

		It("应该正确更新接口状态", func() {
			By("等待状态更新")
			iface := &bizv1.BusinessInterface{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName(ifName, namespace), iface)
				if err != nil {
					return false
				}
				return iface.Status.Phase != ""
			}, timeout, interval).Should(BeTrue())

			By("验证 Phase 状态设置")
			Expect(iface.Status.Phase).To(Or(
				Equal(bizv1.PhasePending),
				Equal(bizv1.PhaseUpdating),
				Equal(bizv1.PhaseReady),
				Equal(bizv1.PhaseFailed),
			))
		})

		It("应该支持更新接口配置", func() {
			By("更新接口描述")
			iface := &bizv1.BusinessInterface{}
			Expect(k8sClient.Get(ctx, NamespacedName(ifName, namespace), iface)).Should(Succeed())

			iface.Spec.Description = "Updated interface description"
			iface.Spec.MTU = 9000
			Expect(k8sClient.Update(ctx, iface)).Should(Succeed())

			By("验证更新生效")
			Eventually(func() string {
				updated := &bizv1.BusinessInterface{}
				err := k8sClient.Get(ctx, NamespacedName(ifName, namespace), updated)
				if err != nil {
					return ""
				}
				return updated.Spec.Description
			}, timeout, interval).Should(Equal("Updated interface description"))

			Eventually(func() uint32 {
				updated := &bizv1.BusinessInterface{}
				err := k8sClient.Get(ctx, NamespacedName(ifName, namespace), updated)
				if err != nil {
					return 0
				}
				return updated.Spec.MTU
			}, timeout, interval).Should(Equal(uint32(9000)))
		})

		It("应该支持删除接口配置", func() {
			By("删除接口资源")
			iface := &bizv1.BusinessInterface{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ifName + "-trunk",
					Namespace: namespace,
				},
			}
			deleteObject(iface)

			By("验证资源已删除")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName(ifName+"-trunk", namespace), &bizv1.BusinessInterface{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
