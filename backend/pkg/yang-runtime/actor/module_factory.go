package actor

import (
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// ModuleFactory creates module-specific ModelActor instances
type ModuleFactory struct {
	clientPool client.ClientPool
}

// NewModuleFactory creates a new ModuleFactory
func NewModuleFactory(clientPool client.ClientPool) *ModuleFactory {
	return &ModuleFactory{
		clientPool: clientPool,
	}
}

// CreateVlanActor creates a ModelActor for VLAN configuration
func (f *ModuleFactory) CreateVlanActor(actorID, deviceID string) *ModelActor[*huawei.HuaweiVlan_Vlan] {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan]()
	actor := NewModelActor[*huawei.HuaweiVlan_Vlan](actorID, deviceID, f.clientPool, translator)
	return actor
}

// CreateIfmActor creates a ModelActor for Interface configuration
func (f *ModuleFactory) CreateIfmActor(actorID, deviceID string) *ModelActor[*huawei.HuaweiIfm_Ifm] {
	translator := NewReflectTranslator[*huawei.HuaweiIfm_Ifm]()
	actor := NewModelActor[*huawei.HuaweiIfm_Ifm](actorID, deviceID, f.clientPool, translator)
	return actor
}

// CreateSystemActor creates a ModelActor for System configuration
func (f *ModuleFactory) CreateSystemActor(actorID, deviceID string) *ModelActor[*huawei.HuaweiSystem_System] {
	translator := NewReflectTranslator[*huawei.HuaweiSystem_System]()
	actor := NewModelActor[*huawei.HuaweiSystem_System](actorID, deviceID, f.clientPool, translator)
	return actor
}

// RegisterAllModules registers all supported modules for a device
func (f *ModuleFactory) RegisterAllModules(deviceActor *DeviceActor) error {
	// Register VLAN module
	vlanActor := f.CreateVlanActor(deviceActor.deviceID+"-vlan", deviceActor.deviceID)
	if err := deviceActor.RegisterModuleActor("vlans", vlanActor); err != nil {
		return err
	}

	// Register Interface module
	ifmActor := f.CreateIfmActor(deviceActor.deviceID+"-ifm", deviceActor.deviceID)
	if err := deviceActor.RegisterModuleActor("interfaces", ifmActor); err != nil {
		return err
	}

	// Register System module
	systemActor := f.CreateSystemActor(deviceActor.deviceID+"-system", deviceActor.deviceID)
	if err := deviceActor.RegisterModuleActor("system", systemActor); err != nil {
		return err
	}

	return nil
}
