package device

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	usmpv1 "github.com/leezesi/usmp/backend/api/core/v1"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// DeviceIPLabel 标记 Device CR 与其凭据 Secret 所属的设备 IP（筛选/反查用）。
const DeviceIPLabel = "usmp.io/device-ip"

// DeviceCRName maps a DeviceID (bare IP) to its CR name（DNS-1123：IPv4 原样，
// 其余字符降级替换）。
func DeviceCRName(ip string) string {
	return strings.ToLower(strings.ReplaceAll(ip, ":", "-"))
}

func deviceSecretName(ip string) string { return "device-cred-" + DeviceCRName(ip) }

// CRDStoreScheme returns the runtime scheme for Device CRs + core resources
// (Secrets)，供 store 与测试/装配共用。
func CRDStoreScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := usmpv1.AddToScheme(s); err != nil {
		return nil, err
	}
	return s, nil
}

// crdStore 是集群模式的 Store 实现（DS-01/04/05）：Device CR 为持久载体，
// 进程内 watch 镜像承接读路径（Get/List 零 apiserver RTT），Put/Delete 写穿。
// 凭据存同 namespace Secret（CR 仅存引用，DS-04）。
type crdStore struct {
	ctx    context.Context
	ns     string
	writer ctrlclient.Client // 直连（写路径 + watch 事件时解析 Secret）

	mu      sync.RWMutex
	devices map[string]client.DeviceConnectionInfo
}

// NewCRDStore 构建集群模式 store：起 Device CR informer 并等待首次同步（重启
// 恢复即来自这次 list）。ctx 取消即停止 watch。同步失败返回错误（调用方降级
// 内存实现，R08）。
func NewCRDStore(ctx context.Context, cfg *rest.Config, namespace string) (Store, error) {
	scheme, err := CRDStoreScheme()
	if err != nil {
		return nil, fmt.Errorf("crdstore scheme: %w", err)
	}
	writer, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("crdstore client: %w", err)
	}
	ca, err := crcache.New(cfg, crcache.Options{
		Scheme:            scheme,
		DefaultNamespaces: map[string]crcache.Config{namespace: {}},
	})
	if err != nil {
		return nil, fmt.Errorf("crdstore cache: %w", err)
	}

	s := &crdStore{
		ctx:     ctx,
		ns:      namespace,
		writer:  writer,
		devices: make(map[string]client.DeviceConnectionInfo),
	}

	inf, err := ca.GetInformer(ctx, &usmpv1.Device{})
	if err != nil {
		return nil, fmt.Errorf("crdstore informer: %w", err)
	}
	if _, err := inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { s.applyEvent(obj) },
		UpdateFunc: func(_, obj interface{}) { s.applyEvent(obj) },
		DeleteFunc: func(obj interface{}) { s.removeEvent(obj) },
	}); err != nil {
		return nil, fmt.Errorf("crdstore event handler: %w", err)
	}

	go func() {
		if err := ca.Start(ctx); err != nil {
			log.Printf("device: crdstore cache stopped: %v", err)
		}
	}()
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !ca.WaitForCacheSync(syncCtx) {
		return nil, fmt.Errorf("crdstore cache sync timeout (namespace %q)", namespace)
	}
	return s, nil
}

// applyEvent 处理 Device CR add/update：spec→连接信息，凭据经 Secret 引用
// 解析（缺失降级空凭据，DS-04 clean fail）。
func (s *crdStore) applyEvent(obj interface{}) {
	dev, ok := obj.(*usmpv1.Device)
	if !ok {
		return
	}
	info := s.resolveInfo(dev)
	s.mu.Lock()
	s.devices[dev.Spec.ManagementIP] = info
	s.mu.Unlock()
}

func (s *crdStore) removeEvent(obj interface{}) {
	if tomb, ok := obj.(toolscache.DeletedFinalStateUnknown); ok {
		obj = tomb.Obj
	}
	dev, ok := obj.(*usmpv1.Device)
	if !ok {
		return
	}
	s.mu.Lock()
	delete(s.devices, dev.Spec.ManagementIP)
	s.mu.Unlock()
}

// resolveInfo 组装完整连接信息：CR spec + Secret 凭据（TLSConfig 不持久化，
// 跨副本恢复为 nil）。
func (s *crdStore) resolveInfo(dev *usmpv1.Device) client.DeviceConnectionInfo {
	info := client.DeviceConnectionInfo{
		IP:       dev.Spec.ManagementIP,
		Port:     dev.Spec.Port,
		Protocol: client.Protocol(dev.Spec.Protocol),
		Timeout:  time.Duration(dev.Spec.TimeoutSeconds) * time.Second,
		Vendor:   dev.Spec.Vendor,
	}
	ref := dev.Spec.CredentialsSecretRef
	if ref == nil {
		return info
	}
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	var sec corev1.Secret
	if err := s.writer.Get(ctx, types.NamespacedName{Namespace: s.ns, Name: ref.Name}, &sec); err != nil {
		// Secret 缺失/不可读：设备仍存在，凭据置空由下游 clean fail（R08）。
		log.Printf("device: credentials secret %q for %s unavailable, degrading to empty credentials: %v",
			ref.Name, dev.Spec.ManagementIP, err)
		return info
	}
	info.Username = string(sec.Data["username"])
	info.Password = string(sec.Data["password"])
	return info
}

func (s *crdStore) Get(id string) (client.DeviceConnectionInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.devices[id]
	return info, ok
}

func (s *crdStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.devices))
	for id := range s.devices {
		ids = append(ids, id)
	}
	return ids
}

// Put 写穿：先 upsert 凭据 Secret 再 upsert Device CR（D1 写序，存在性以 CR
// 为权威），成功后同步更新本地镜像（写后立即可读，不等 watch 回流）。
func (s *crdStore) Put(id string, info client.DeviceConnectionInfo) error {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	secName := deviceSecretName(id)
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secName,
			Namespace: s.ns,
			Labels:    map[string]string{DeviceIPLabel: id},
		},
		Data: map[string][]byte{
			"username": []byte(info.Username),
			"password": []byte(info.Password),
		},
	}
	if err := s.upsert(ctx, sec, func(existing ctrlclient.Object) {
		existing.(*corev1.Secret).Data = sec.Data
	}); err != nil {
		return fmt.Errorf("persist credentials secret for %s: %w", id, err)
	}

	dev := &usmpv1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeviceCRName(id),
			Namespace: s.ns,
			Labels:    map[string]string{DeviceIPLabel: id},
		},
		Spec: usmpv1.DeviceSpec{
			ManagementIP:         id,
			Port:                 info.Port,
			Protocol:             string(info.Protocol),
			TimeoutSeconds:       int(info.Timeout / time.Second),
			Vendor:               info.Vendor,
			CredentialsSecretRef: &usmpv1.LocalSecretRef{Name: secName},
		},
	}
	if err := s.upsert(ctx, dev, func(existing ctrlclient.Object) {
		existing.(*usmpv1.Device).Spec = dev.Spec
	}); err != nil {
		return fmt.Errorf("persist device CR for %s: %w", id, err)
	}

	s.mu.Lock()
	s.devices[id] = info
	s.mu.Unlock()
	return nil
}

// upsert 创建对象，已存在则读回后经 mutate 更新（一次冲突重读重试）。
func (s *crdStore) upsert(ctx context.Context, obj ctrlclient.Object, mutate func(existing ctrlclient.Object)) error {
	err := s.writer.Create(ctx, obj)
	if err == nil || !apierrors.IsAlreadyExists(err) {
		return err
	}
	for attempt := 0; attempt < 2; attempt++ {
		existing := obj.DeepCopyObject().(ctrlclient.Object)
		if err := s.writer.Get(ctx, ctrlclient.ObjectKeyFromObject(obj), existing); err != nil {
			return err
		}
		mutate(existing)
		err = s.writer.Update(ctx, existing)
		if err == nil || !apierrors.IsConflict(err) {
			return err
		}
	}
	return err
}

// Delete 反向清理：先删 Device CR 再删凭据 Secret（NotFound 容忍幂等），
// 成功后移除本地镜像。
func (s *crdStore) Delete(id string) error {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	dev := &usmpv1.Device{}
	dev.Namespace, dev.Name = s.ns, DeviceCRName(id)
	if err := s.writer.Delete(ctx, dev); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete device CR for %s: %w", id, err)
	}
	sec := &corev1.Secret{}
	sec.Namespace, sec.Name = s.ns, deviceSecretName(id)
	if err := s.writer.Delete(ctx, sec); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete credentials secret for %s: %w", id, err)
	}

	s.mu.Lock()
	delete(s.devices, id)
	s.mu.Unlock()
	return nil
}

// Persistent 标记本 store 内容跨进程重启存活（DS-03 集群模式判据）。
func (s *crdStore) Persistent() bool { return true }

// IsPersistent reports whether the store's contents survive restarts（集群
// 模式下 DeviceHandler 据此忽略 USMP_SEED_DEVICE，设备集合仅来自 CR）。
func IsPersistent(s Store) bool {
	p, ok := s.(interface{ Persistent() bool })
	return ok && p.Persistent()
}
