package huawei

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

var _ = fmt.Sprintf("")
var _ = json.Marshal
var _ = reflect.TypeOf
var _ = yang.Entry{}
var _ = ytypes.Schema{}

type E_HuaweiIfm_ClassType int64

// IsYANGGoEnum ensures that HuaweiIfm_ClassType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_ClassType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_ClassType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_ClassType.
func (E_HuaweiIfm_ClassType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_ClassType.
func (e E_HuaweiIfm_ClassType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_ClassType")
}

const (
	// HuaweiIfm_ClassType_UNSET corresponds to the value UNSET of HuaweiIfm_ClassType
	HuaweiIfm_ClassType_UNSET E_HuaweiIfm_ClassType = 0
	// HuaweiIfm_ClassType_main_interface corresponds to the value main_interface of HuaweiIfm_ClassType
	HuaweiIfm_ClassType_main_interface E_HuaweiIfm_ClassType = 1
	// HuaweiIfm_ClassType_sub_interface corresponds to the value sub_interface of HuaweiIfm_ClassType
	HuaweiIfm_ClassType_sub_interface E_HuaweiIfm_ClassType = 2
)

// E_HuaweiIfm_DampLevelType is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_DampLevelType. An additional value named
// HuaweiIfm_DampLevelType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_DampLevelType int64

// IsYANGGoEnum ensures that HuaweiIfm_DampLevelType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_DampLevelType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_DampLevelType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_DampLevelType.
func (E_HuaweiIfm_DampLevelType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_DampLevelType.
func (e E_HuaweiIfm_DampLevelType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_DampLevelType")
}

const (
	// HuaweiIfm_DampLevelType_UNSET corresponds to the value UNSET of HuaweiIfm_DampLevelType
	HuaweiIfm_DampLevelType_UNSET E_HuaweiIfm_DampLevelType = 0
	// HuaweiIfm_DampLevelType_light corresponds to the value light of HuaweiIfm_DampLevelType
	HuaweiIfm_DampLevelType_light E_HuaweiIfm_DampLevelType = 1
	// HuaweiIfm_DampLevelType_middle corresponds to the value middle of HuaweiIfm_DampLevelType
	HuaweiIfm_DampLevelType_middle E_HuaweiIfm_DampLevelType = 2
	// HuaweiIfm_DampLevelType_heavy corresponds to the value heavy of HuaweiIfm_DampLevelType
	HuaweiIfm_DampLevelType_heavy E_HuaweiIfm_DampLevelType = 3
)

// E_HuaweiIfm_DampStatusType is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_DampStatusType. An additional value named
// HuaweiIfm_DampStatusType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_DampStatusType int64

// IsYANGGoEnum ensures that HuaweiIfm_DampStatusType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_DampStatusType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_DampStatusType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_DampStatusType.
func (E_HuaweiIfm_DampStatusType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_DampStatusType.
func (e E_HuaweiIfm_DampStatusType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_DampStatusType")
}

const (
	// HuaweiIfm_DampStatusType_UNSET corresponds to the value UNSET of HuaweiIfm_DampStatusType
	HuaweiIfm_DampStatusType_UNSET E_HuaweiIfm_DampStatusType = 0
	// HuaweiIfm_DampStatusType_suppressed corresponds to the value suppressed of HuaweiIfm_DampStatusType
	HuaweiIfm_DampStatusType_suppressed E_HuaweiIfm_DampStatusType = 1
	// HuaweiIfm_DampStatusType_unsuppressed corresponds to the value unsuppressed of HuaweiIfm_DampStatusType
	HuaweiIfm_DampStatusType_unsuppressed E_HuaweiIfm_DampStatusType = 2
)

// E_HuaweiIfm_EncapsulationType is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_EncapsulationType. An additional value named
// HuaweiIfm_EncapsulationType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_EncapsulationType int64

// IsYANGGoEnum ensures that HuaweiIfm_EncapsulationType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_EncapsulationType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_EncapsulationType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_EncapsulationType.
func (E_HuaweiIfm_EncapsulationType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_EncapsulationType.
func (e E_HuaweiIfm_EncapsulationType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_EncapsulationType")
}

const (
	// HuaweiIfm_EncapsulationType_UNSET corresponds to the value UNSET of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_UNSET E_HuaweiIfm_EncapsulationType = 0
	// HuaweiIfm_EncapsulationType_vlan_type corresponds to the value vlan_type of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_vlan_type E_HuaweiIfm_EncapsulationType = 1
	// HuaweiIfm_EncapsulationType_dot1q corresponds to the value dot1q of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_dot1q E_HuaweiIfm_EncapsulationType = 2
	// HuaweiIfm_EncapsulationType_qinq corresponds to the value qinq of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_qinq E_HuaweiIfm_EncapsulationType = 3
	// HuaweiIfm_EncapsulationType_p2p corresponds to the value p2p of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_p2p E_HuaweiIfm_EncapsulationType = 4
	// HuaweiIfm_EncapsulationType_p2mp corresponds to the value p2mp of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_p2mp E_HuaweiIfm_EncapsulationType = 5
	// HuaweiIfm_EncapsulationType_l2ve corresponds to the value l2ve of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_l2ve E_HuaweiIfm_EncapsulationType = 6
	// HuaweiIfm_EncapsulationType_l3ve corresponds to the value l3ve of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_l3ve E_HuaweiIfm_EncapsulationType = 7
	// HuaweiIfm_EncapsulationType_vlan_type_policy corresponds to the value vlan_type_policy of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_vlan_type_policy E_HuaweiIfm_EncapsulationType = 8
	// HuaweiIfm_EncapsulationType_dot1q_policy corresponds to the value dot1q_policy of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_dot1q_policy E_HuaweiIfm_EncapsulationType = 9
	// HuaweiIfm_EncapsulationType_stacking_policy corresponds to the value stacking_policy of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_stacking_policy E_HuaweiIfm_EncapsulationType = 10
	// HuaweiIfm_EncapsulationType_untag_policy corresponds to the value untag_policy of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_untag_policy E_HuaweiIfm_EncapsulationType = 11
	// HuaweiIfm_EncapsulationType_qinq_mapping corresponds to the value qinq_mapping of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_qinq_mapping E_HuaweiIfm_EncapsulationType = 12
	// HuaweiIfm_EncapsulationType_l2vc corresponds to the value l2vc of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_l2vc E_HuaweiIfm_EncapsulationType = 13
	// HuaweiIfm_EncapsulationType_l3vc corresponds to the value l3vc of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_l3vc E_HuaweiIfm_EncapsulationType = 14
	// HuaweiIfm_EncapsulationType_evc_untag corresponds to the value evc_untag of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_evc_untag E_HuaweiIfm_EncapsulationType = 15
	// HuaweiIfm_EncapsulationType_evc_dot1q corresponds to the value evc_dot1q of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_evc_dot1q E_HuaweiIfm_EncapsulationType = 16
	// HuaweiIfm_EncapsulationType_evc_qinq corresponds to the value evc_qinq of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_evc_qinq E_HuaweiIfm_EncapsulationType = 17
	// HuaweiIfm_EncapsulationType_evc_default corresponds to the value evc_default of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_evc_default E_HuaweiIfm_EncapsulationType = 18
	// HuaweiIfm_EncapsulationType_evc_dot1q_policy corresponds to the value evc_dot1q_policy of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_evc_dot1q_policy E_HuaweiIfm_EncapsulationType = 19
	// HuaweiIfm_EncapsulationType_ietf corresponds to the value ietf of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_ietf E_HuaweiIfm_EncapsulationType = 20
	// HuaweiIfm_EncapsulationType_nonstandard corresponds to the value nonstandard of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_nonstandard E_HuaweiIfm_EncapsulationType = 21
	// HuaweiIfm_EncapsulationType_user_vlan corresponds to the value user_vlan of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_user_vlan E_HuaweiIfm_EncapsulationType = 22
	// HuaweiIfm_EncapsulationType_user_vlan_anyother corresponds to the value user_vlan_anyother of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_user_vlan_anyother E_HuaweiIfm_EncapsulationType = 23
	// HuaweiIfm_EncapsulationType_qin_link corresponds to the value qin_link of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_qin_link E_HuaweiIfm_EncapsulationType = 24
	// HuaweiIfm_EncapsulationType_soft_gre_ve corresponds to the value soft_gre_ve of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_soft_gre_ve E_HuaweiIfm_EncapsulationType = 25
	// HuaweiIfm_EncapsulationType_l3ve_ter corresponds to the value l3ve_ter of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_l3ve_ter E_HuaweiIfm_EncapsulationType = 26
	// HuaweiIfm_EncapsulationType_l3ve_acc corresponds to the value l3ve_acc of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_l3ve_acc E_HuaweiIfm_EncapsulationType = 27
	// HuaweiIfm_EncapsulationType_invalid corresponds to the value invalid of HuaweiIfm_EncapsulationType
	HuaweiIfm_EncapsulationType_invalid E_HuaweiIfm_EncapsulationType = 256
)

// E_HuaweiIfm_ErrorDownType is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_ErrorDownType. An additional value named
// HuaweiIfm_ErrorDownType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_ErrorDownType int64

// IsYANGGoEnum ensures that HuaweiIfm_ErrorDownType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_ErrorDownType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_ErrorDownType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_ErrorDownType.
func (E_HuaweiIfm_ErrorDownType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_ErrorDownType.
func (e E_HuaweiIfm_ErrorDownType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_ErrorDownType")
}

const (
	// HuaweiIfm_ErrorDownType_UNSET corresponds to the value UNSET of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_UNSET E_HuaweiIfm_ErrorDownType = 0
	// HuaweiIfm_ErrorDownType_bpdu_protection corresponds to the value bpdu_protection of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_bpdu_protection E_HuaweiIfm_ErrorDownType = 1
	// HuaweiIfm_ErrorDownType_auto_defend corresponds to the value auto_defend of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_auto_defend E_HuaweiIfm_ErrorDownType = 2
	// HuaweiIfm_ErrorDownType_monitor_link corresponds to the value monitor_link of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_monitor_link E_HuaweiIfm_ErrorDownType = 3
	// HuaweiIfm_ErrorDownType_portsec_reached_limit corresponds to the value portsec_reached_limit of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_portsec_reached_limit E_HuaweiIfm_ErrorDownType = 66
	// HuaweiIfm_ErrorDownType_storm_control corresponds to the value storm_control of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_storm_control E_HuaweiIfm_ErrorDownType = 67
	// HuaweiIfm_ErrorDownType_loopback_detect corresponds to the value loopback_detect of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_loopback_detect E_HuaweiIfm_ErrorDownType = 68
	// HuaweiIfm_ErrorDownType_dual_active corresponds to the value dual_active of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_dual_active E_HuaweiIfm_ErrorDownType = 69
	// HuaweiIfm_ErrorDownType_mac_address_flapping corresponds to the value mac_address_flapping of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_mac_address_flapping E_HuaweiIfm_ErrorDownType = 70
	// HuaweiIfm_ErrorDownType_no_stack_link corresponds to the value no_stack_link of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_no_stack_link E_HuaweiIfm_ErrorDownType = 71
	// HuaweiIfm_ErrorDownType_crc_statistics corresponds to the value crc_statistics of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_crc_statistics E_HuaweiIfm_ErrorDownType = 72
	// HuaweiIfm_ErrorDownType_transceiver_power_low corresponds to the value transceiver_power_low of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_transceiver_power_low E_HuaweiIfm_ErrorDownType = 73
	// HuaweiIfm_ErrorDownType_link_flap corresponds to the value link_flap of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_link_flap E_HuaweiIfm_ErrorDownType = 74
	// HuaweiIfm_ErrorDownType_l2_loop_occured corresponds to the value l2_loop_occured of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_l2_loop_occured E_HuaweiIfm_ErrorDownType = 75
	// HuaweiIfm_ErrorDownType_stack_member_exceed_limit corresponds to the value stack_member_exceed_limit of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_stack_member_exceed_limit E_HuaweiIfm_ErrorDownType = 76
	// HuaweiIfm_ErrorDownType_spine_member_exceed_limit corresponds to the value spine_member_exceed_limit of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_spine_member_exceed_limit E_HuaweiIfm_ErrorDownType = 77
	// HuaweiIfm_ErrorDownType_resource_mismatch corresponds to the value resource_mismatch of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_resource_mismatch E_HuaweiIfm_ErrorDownType = 78
	// HuaweiIfm_ErrorDownType_leaf_mstp corresponds to the value leaf_mstp of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_leaf_mstp E_HuaweiIfm_ErrorDownType = 79
	// HuaweiIfm_ErrorDownType_m_lag corresponds to the value m_lag of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_m_lag E_HuaweiIfm_ErrorDownType = 80
	// HuaweiIfm_ErrorDownType_fabric_uplink_threshold corresponds to the value fabric_uplink_threshold of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_fabric_uplink_threshold E_HuaweiIfm_ErrorDownType = 81
	// HuaweiIfm_ErrorDownType_stack_config_conflict corresponds to the value stack_config_conflict of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_stack_config_conflict E_HuaweiIfm_ErrorDownType = 82
	// HuaweiIfm_ErrorDownType_spine_type_unsupported corresponds to the value spine_type_unsupported of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_spine_type_unsupported E_HuaweiIfm_ErrorDownType = 83
	// HuaweiIfm_ErrorDownType_stack_packet_defensive corresponds to the value stack_packet_defensive of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_stack_packet_defensive E_HuaweiIfm_ErrorDownType = 84
	// HuaweiIfm_ErrorDownType_forward_engine_buffer_failed corresponds to the value forward_engine_buffer_failed of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_forward_engine_buffer_failed E_HuaweiIfm_ErrorDownType = 86
	// HuaweiIfm_ErrorDownType_forward_engine_interface_failed corresponds to the value forward_engine_interface_failed of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_forward_engine_interface_failed E_HuaweiIfm_ErrorDownType = 87
	// HuaweiIfm_ErrorDownType_fabric_link_failure corresponds to the value fabric_link_failure of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_fabric_link_failure E_HuaweiIfm_ErrorDownType = 88
	// HuaweiIfm_ErrorDownType_m_lag_consistency_check corresponds to the value m_lag_consistency_check of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_m_lag_consistency_check E_HuaweiIfm_ErrorDownType = 89
	// HuaweiIfm_ErrorDownType_pfc_deadlock corresponds to the value pfc_deadlock of HuaweiIfm_ErrorDownType
	HuaweiIfm_ErrorDownType_pfc_deadlock E_HuaweiIfm_ErrorDownType = 90
)

// E_HuaweiIfm_LinkProtocol is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_LinkProtocol. An additional value named
// HuaweiIfm_LinkProtocol_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_LinkProtocol int64

// IsYANGGoEnum ensures that HuaweiIfm_LinkProtocol implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_LinkProtocol can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_LinkProtocol) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_LinkProtocol.
func (E_HuaweiIfm_LinkProtocol) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_LinkProtocol.
func (e E_HuaweiIfm_LinkProtocol) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_LinkProtocol")
}

const (
	// HuaweiIfm_LinkProtocol_UNSET corresponds to the value UNSET of HuaweiIfm_LinkProtocol
	HuaweiIfm_LinkProtocol_UNSET E_HuaweiIfm_LinkProtocol = 0
	// HuaweiIfm_LinkProtocol_ethernet corresponds to the value ethernet of HuaweiIfm_LinkProtocol
	HuaweiIfm_LinkProtocol_ethernet E_HuaweiIfm_LinkProtocol = 1
	// HuaweiIfm_LinkProtocol_ppp corresponds to the value ppp of HuaweiIfm_LinkProtocol
	HuaweiIfm_LinkProtocol_ppp E_HuaweiIfm_LinkProtocol = 2
	// HuaweiIfm_LinkProtocol_hdlc corresponds to the value hdlc of HuaweiIfm_LinkProtocol
	HuaweiIfm_LinkProtocol_hdlc E_HuaweiIfm_LinkProtocol = 3
	// HuaweiIfm_LinkProtocol_fr corresponds to the value fr of HuaweiIfm_LinkProtocol
	HuaweiIfm_LinkProtocol_fr E_HuaweiIfm_LinkProtocol = 4
	// HuaweiIfm_LinkProtocol_atm corresponds to the value atm of HuaweiIfm_LinkProtocol
	HuaweiIfm_LinkProtocol_atm E_HuaweiIfm_LinkProtocol = 7
	// HuaweiIfm_LinkProtocol_tdm corresponds to the value tdm of HuaweiIfm_LinkProtocol
	HuaweiIfm_LinkProtocol_tdm E_HuaweiIfm_LinkProtocol = 8
)

// E_HuaweiIfm_LinkQualityGradeType is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_LinkQualityGradeType. An additional value named
// HuaweiIfm_LinkQualityGradeType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_LinkQualityGradeType int64

// IsYANGGoEnum ensures that HuaweiIfm_LinkQualityGradeType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_LinkQualityGradeType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_LinkQualityGradeType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_LinkQualityGradeType.
func (E_HuaweiIfm_LinkQualityGradeType) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	return ΛEnum
}

// String returns a logging-friendly string for E_HuaweiIfm_LinkQualityGradeType.
func (e E_HuaweiIfm_LinkQualityGradeType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_LinkQualityGradeType")
}

const (
	// HuaweiIfm_LinkQualityGradeType_UNSET corresponds to the value UNSET of HuaweiIfm_LinkQualityGradeType
	HuaweiIfm_LinkQualityGradeType_UNSET E_HuaweiIfm_LinkQualityGradeType = 0
	// HuaweiIfm_LinkQualityGradeType_good corresponds to the value good of HuaweiIfm_LinkQualityGradeType
	HuaweiIfm_LinkQualityGradeType_good E_HuaweiIfm_LinkQualityGradeType = 1
	// HuaweiIfm_LinkQualityGradeType_high corresponds to the value high of HuaweiIfm_LinkQualityGradeType
	HuaweiIfm_LinkQualityGradeType_high E_HuaweiIfm_LinkQualityGradeType = 2
	// HuaweiIfm_LinkQualityGradeType_middle corresponds to the value middle of HuaweiIfm_LinkQualityGradeType
	HuaweiIfm_LinkQualityGradeType_middle E_HuaweiIfm_LinkQualityGradeType = 3
	// HuaweiIfm_LinkQualityGradeType_low corresponds to the value low of HuaweiIfm_LinkQualityGradeType
	HuaweiIfm_LinkQualityGradeType_low E_HuaweiIfm_LinkQualityGradeType = 5
)

// E_HuaweiIfm_NetworkLayerState is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_NetworkLayerState. An additional value named
// HuaweiIfm_NetworkLayerState_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_NetworkLayerState int64

// IsYANGGoEnum ensures that HuaweiIfm_NetworkLayerState implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_NetworkLayerState can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_NetworkLayerState) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_NetworkLayerState.
func (E_HuaweiIfm_NetworkLayerState) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_NetworkLayerState.
func (e E_HuaweiIfm_NetworkLayerState) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_NetworkLayerState")
}

const (
	// HuaweiIfm_NetworkLayerState_UNSET corresponds to the value UNSET of HuaweiIfm_NetworkLayerState
	HuaweiIfm_NetworkLayerState_UNSET E_HuaweiIfm_NetworkLayerState = 0
	// HuaweiIfm_NetworkLayerState_ipv4_ipv6_up corresponds to the value ipv4_ipv6_up of HuaweiIfm_NetworkLayerState
	HuaweiIfm_NetworkLayerState_ipv4_ipv6_up E_HuaweiIfm_NetworkLayerState = 1
	// HuaweiIfm_NetworkLayerState_ipv4_ipv6_down corresponds to the value ipv4_ipv6_down of HuaweiIfm_NetworkLayerState
	HuaweiIfm_NetworkLayerState_ipv4_ipv6_down E_HuaweiIfm_NetworkLayerState = 196609
)

// E_HuaweiIfm_PortStatus is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_PortStatus. An additional value named
// HuaweiIfm_PortStatus_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_PortStatus int64

// IsYANGGoEnum ensures that HuaweiIfm_PortStatus implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_PortStatus can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_PortStatus) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_PortStatus.
func (E_HuaweiIfm_PortStatus) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_PortStatus.
func (e E_HuaweiIfm_PortStatus) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_PortStatus")
}

const (
	// HuaweiIfm_PortStatus_UNSET corresponds to the value UNSET of HuaweiIfm_PortStatus
	HuaweiIfm_PortStatus_UNSET E_HuaweiIfm_PortStatus = 0
	// HuaweiIfm_PortStatus_down corresponds to the value down of HuaweiIfm_PortStatus
	HuaweiIfm_PortStatus_down E_HuaweiIfm_PortStatus = 1
	// HuaweiIfm_PortStatus_up corresponds to the value up of HuaweiIfm_PortStatus
	HuaweiIfm_PortStatus_up E_HuaweiIfm_PortStatus = 2
)

// E_HuaweiIfm_PortType is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_PortType. An additional value named
// HuaweiIfm_PortType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_PortType int64

// IsYANGGoEnum ensures that HuaweiIfm_PortType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_PortType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_PortType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_PortType.
func (E_HuaweiIfm_PortType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_PortType.
func (e E_HuaweiIfm_PortType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_PortType")
}

const (
	// HuaweiIfm_PortType_UNSET corresponds to the value UNSET of HuaweiIfm_PortType
	HuaweiIfm_PortType_UNSET E_HuaweiIfm_PortType = 0
	// HuaweiIfm_PortType_Ethernet corresponds to the value Ethernet of HuaweiIfm_PortType
	HuaweiIfm_PortType_Ethernet E_HuaweiIfm_PortType = 1
	// HuaweiIfm_PortType_GigabitEthernet corresponds to the value GigabitEthernet of HuaweiIfm_PortType
	HuaweiIfm_PortType_GigabitEthernet E_HuaweiIfm_PortType = 3
	// HuaweiIfm_PortType_Eth_Trunk corresponds to the value Eth_Trunk of HuaweiIfm_PortType
	HuaweiIfm_PortType_Eth_Trunk E_HuaweiIfm_PortType = 5
	// HuaweiIfm_PortType_Ip_Trunk corresponds to the value Ip_Trunk of HuaweiIfm_PortType
	HuaweiIfm_PortType_Ip_Trunk E_HuaweiIfm_PortType = 6
	// HuaweiIfm_PortType_Virtual_Ethernet corresponds to the value Virtual_Ethernet of HuaweiIfm_PortType
	HuaweiIfm_PortType_Virtual_Ethernet E_HuaweiIfm_PortType = 7
	// HuaweiIfm_PortType_Serial corresponds to the value Serial of HuaweiIfm_PortType
	HuaweiIfm_PortType_Serial E_HuaweiIfm_PortType = 9
	// HuaweiIfm_PortType_Pos corresponds to the value Pos of HuaweiIfm_PortType
	HuaweiIfm_PortType_Pos E_HuaweiIfm_PortType = 10
	// HuaweiIfm_PortType_Cpos corresponds to the value Cpos of HuaweiIfm_PortType
	HuaweiIfm_PortType_Cpos E_HuaweiIfm_PortType = 11
	// HuaweiIfm_PortType_ATM corresponds to the value ATM of HuaweiIfm_PortType
	HuaweiIfm_PortType_ATM E_HuaweiIfm_PortType = 12
	// HuaweiIfm_PortType_Tunnel corresponds to the value Tunnel of HuaweiIfm_PortType
	HuaweiIfm_PortType_Tunnel E_HuaweiIfm_PortType = 15
	// HuaweiIfm_PortType_Vlanif corresponds to the value Vlanif of HuaweiIfm_PortType
	HuaweiIfm_PortType_Vlanif E_HuaweiIfm_PortType = 16
	// HuaweiIfm_PortType_NULL corresponds to the value NULL of HuaweiIfm_PortType
	HuaweiIfm_PortType_NULL E_HuaweiIfm_PortType = 19
	// HuaweiIfm_PortType_LoopBack corresponds to the value LoopBack of HuaweiIfm_PortType
	HuaweiIfm_PortType_LoopBack E_HuaweiIfm_PortType = 20
	// HuaweiIfm_PortType_100GE corresponds to the value 100GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_100GE E_HuaweiIfm_PortType = 21
	// HuaweiIfm_PortType_Lmpif corresponds to the value Lmpif of HuaweiIfm_PortType
	HuaweiIfm_PortType_Lmpif E_HuaweiIfm_PortType = 22
	// HuaweiIfm_PortType_MTunnel corresponds to the value MTunnel of HuaweiIfm_PortType
	HuaweiIfm_PortType_MTunnel E_HuaweiIfm_PortType = 23
	// HuaweiIfm_PortType_40GE corresponds to the value 40GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_40GE E_HuaweiIfm_PortType = 24
	// HuaweiIfm_PortType_10GE corresponds to the value 10GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_10GE E_HuaweiIfm_PortType = 25
	// HuaweiIfm_PortType_GEBrief corresponds to the value GEBrief of HuaweiIfm_PortType
	HuaweiIfm_PortType_GEBrief E_HuaweiIfm_PortType = 26
	// HuaweiIfm_PortType_MEth corresponds to the value MEth of HuaweiIfm_PortType
	HuaweiIfm_PortType_MEth E_HuaweiIfm_PortType = 27
	// HuaweiIfm_PortType_Stack_Port corresponds to the value Stack_Port of HuaweiIfm_PortType
	HuaweiIfm_PortType_Stack_Port E_HuaweiIfm_PortType = 28
	// HuaweiIfm_PortType_Sip corresponds to the value Sip of HuaweiIfm_PortType
	HuaweiIfm_PortType_Sip E_HuaweiIfm_PortType = 29
	// HuaweiIfm_PortType_E1 corresponds to the value E1 of HuaweiIfm_PortType
	HuaweiIfm_PortType_E1 E_HuaweiIfm_PortType = 31
	// HuaweiIfm_PortType_Mp_group corresponds to the value Mp_group of HuaweiIfm_PortType
	HuaweiIfm_PortType_Mp_group E_HuaweiIfm_PortType = 32
	// HuaweiIfm_PortType_Ima_group corresponds to the value Ima_group of HuaweiIfm_PortType
	HuaweiIfm_PortType_Ima_group E_HuaweiIfm_PortType = 33
	// HuaweiIfm_PortType_VMEth corresponds to the value VMEth of HuaweiIfm_PortType
	HuaweiIfm_PortType_VMEth E_HuaweiIfm_PortType = 34
	// HuaweiIfm_PortType_Remote_Ap corresponds to the value Remote_Ap of HuaweiIfm_PortType
	HuaweiIfm_PortType_Remote_Ap E_HuaweiIfm_PortType = 36
	// HuaweiIfm_PortType_VBridge corresponds to the value VBridge of HuaweiIfm_PortType
	HuaweiIfm_PortType_VBridge E_HuaweiIfm_PortType = 37
	// HuaweiIfm_PortType_Atm_Bundle corresponds to the value Atm_Bundle of HuaweiIfm_PortType
	HuaweiIfm_PortType_Atm_Bundle E_HuaweiIfm_PortType = 38
	// HuaweiIfm_PortType_Fiber_Channel corresponds to the value Fiber_Channel of HuaweiIfm_PortType
	HuaweiIfm_PortType_Fiber_Channel E_HuaweiIfm_PortType = 39
	// HuaweiIfm_PortType_Infiniband corresponds to the value Infiniband of HuaweiIfm_PortType
	HuaweiIfm_PortType_Infiniband E_HuaweiIfm_PortType = 40
	// HuaweiIfm_PortType_Vbdif corresponds to the value Vbdif of HuaweiIfm_PortType
	HuaweiIfm_PortType_Vbdif E_HuaweiIfm_PortType = 41
	// HuaweiIfm_PortType_T1 corresponds to the value T1 of HuaweiIfm_PortType
	HuaweiIfm_PortType_T1 E_HuaweiIfm_PortType = 42
	// HuaweiIfm_PortType_T3 corresponds to the value T3 of HuaweiIfm_PortType
	HuaweiIfm_PortType_T3 E_HuaweiIfm_PortType = 43
	// HuaweiIfm_PortType_VC4 corresponds to the value VC4 of HuaweiIfm_PortType
	HuaweiIfm_PortType_VC4 E_HuaweiIfm_PortType = 44
	// HuaweiIfm_PortType_VC12 corresponds to the value VC12 of HuaweiIfm_PortType
	HuaweiIfm_PortType_VC12 E_HuaweiIfm_PortType = 45
	// HuaweiIfm_PortType_Global_VE corresponds to the value Global_VE of HuaweiIfm_PortType
	HuaweiIfm_PortType_Global_VE E_HuaweiIfm_PortType = 46
	// HuaweiIfm_PortType_Fabric_Port corresponds to the value Fabric_Port of HuaweiIfm_PortType
	HuaweiIfm_PortType_Fabric_Port E_HuaweiIfm_PortType = 47
	// HuaweiIfm_PortType_E3 corresponds to the value E3 of HuaweiIfm_PortType
	HuaweiIfm_PortType_E3 E_HuaweiIfm_PortType = 49
	// HuaweiIfm_PortType_Vp corresponds to the value Vp of HuaweiIfm_PortType
	HuaweiIfm_PortType_Vp E_HuaweiIfm_PortType = 50
	// HuaweiIfm_PortType_DcnInterface corresponds to the value DcnInterface of HuaweiIfm_PortType
	HuaweiIfm_PortType_DcnInterface E_HuaweiIfm_PortType = 51
	// HuaweiIfm_PortType_Cpos_Trunk corresponds to the value Cpos_Trunk of HuaweiIfm_PortType
	HuaweiIfm_PortType_Cpos_Trunk E_HuaweiIfm_PortType = 52
	// HuaweiIfm_PortType_Trunk_Serial corresponds to the value Trunk_Serial of HuaweiIfm_PortType
	HuaweiIfm_PortType_Trunk_Serial E_HuaweiIfm_PortType = 53
	// HuaweiIfm_PortType_Global_Mp_Group corresponds to the value Global_Mp_Group of HuaweiIfm_PortType
	HuaweiIfm_PortType_Global_Mp_Group E_HuaweiIfm_PortType = 54
	// HuaweiIfm_PortType_Otn corresponds to the value Otn of HuaweiIfm_PortType
	HuaweiIfm_PortType_Otn E_HuaweiIfm_PortType = 56
	// HuaweiIfm_PortType_Global_Ima_Group corresponds to the value Global_Ima_Group of HuaweiIfm_PortType
	HuaweiIfm_PortType_Global_Ima_Group E_HuaweiIfm_PortType = 58
	// HuaweiIfm_PortType_Pos_Trunk corresponds to the value Pos_Trunk of HuaweiIfm_PortType
	HuaweiIfm_PortType_Pos_Trunk E_HuaweiIfm_PortType = 60
	// HuaweiIfm_PortType_Gmpls_Uni corresponds to the value Gmpls_Uni of HuaweiIfm_PortType
	HuaweiIfm_PortType_Gmpls_Uni E_HuaweiIfm_PortType = 64
	// HuaweiIfm_PortType_Wdm corresponds to the value Wdm of HuaweiIfm_PortType
	HuaweiIfm_PortType_Wdm E_HuaweiIfm_PortType = 65
	// HuaweiIfm_PortType_Nve corresponds to the value Nve of HuaweiIfm_PortType
	HuaweiIfm_PortType_Nve E_HuaweiIfm_PortType = 66
	// HuaweiIfm_PortType_FCoE_Port corresponds to the value FCoE_Port of HuaweiIfm_PortType
	HuaweiIfm_PortType_FCoE_Port E_HuaweiIfm_PortType = 68
	// HuaweiIfm_PortType_Virtual_Template corresponds to the value Virtual_Template of HuaweiIfm_PortType
	HuaweiIfm_PortType_Virtual_Template E_HuaweiIfm_PortType = 69
	// HuaweiIfm_PortType_FC corresponds to the value FC of HuaweiIfm_PortType
	HuaweiIfm_PortType_FC E_HuaweiIfm_PortType = 71
	// HuaweiIfm_PortType_4x10GE corresponds to the value 4x10GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_4x10GE E_HuaweiIfm_PortType = 72
	// HuaweiIfm_PortType_10x10GE corresponds to the value 10x10GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_10x10GE E_HuaweiIfm_PortType = 73
	// HuaweiIfm_PortType_3x40GE corresponds to the value 3x40GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_3x40GE E_HuaweiIfm_PortType = 74
	// HuaweiIfm_PortType_4x25GE corresponds to the value 4x25GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_4x25GE E_HuaweiIfm_PortType = 75
	// HuaweiIfm_PortType_25GE corresponds to the value 25GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_25GE E_HuaweiIfm_PortType = 76
	// HuaweiIfm_PortType_IMEth corresponds to the value IMEth of HuaweiIfm_PortType
	HuaweiIfm_PortType_IMEth E_HuaweiIfm_PortType = 80
	// HuaweiIfm_PortType_PW_VE corresponds to the value PW_VE of HuaweiIfm_PortType
	HuaweiIfm_PortType_PW_VE E_HuaweiIfm_PortType = 89
	// HuaweiIfm_PortType_VX_Tunnel corresponds to the value VX_Tunnel of HuaweiIfm_PortType
	HuaweiIfm_PortType_VX_Tunnel E_HuaweiIfm_PortType = 90
	// HuaweiIfm_PortType_ServiceIf corresponds to the value ServiceIf of HuaweiIfm_PortType
	HuaweiIfm_PortType_ServiceIf E_HuaweiIfm_PortType = 91
	// HuaweiIfm_PortType_XGigabitEthernet corresponds to the value XGigabitEthernet of HuaweiIfm_PortType
	HuaweiIfm_PortType_XGigabitEthernet E_HuaweiIfm_PortType = 92
	// HuaweiIfm_PortType_200GE corresponds to the value 200GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_200GE E_HuaweiIfm_PortType = 93
	// HuaweiIfm_PortType_Virtual_ODUk corresponds to the value Virtual_ODUk of HuaweiIfm_PortType
	HuaweiIfm_PortType_Virtual_ODUk E_HuaweiIfm_PortType = 95
	// HuaweiIfm_PortType_FlexE corresponds to the value FlexE of HuaweiIfm_PortType
	HuaweiIfm_PortType_FlexE E_HuaweiIfm_PortType = 96
	// HuaweiIfm_PortType_FlexE_200GE corresponds to the value FlexE_200GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_FlexE_200GE E_HuaweiIfm_PortType = 97
	// HuaweiIfm_PortType_50_OR_100GE corresponds to the value 50|100GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_50_OR_100GE E_HuaweiIfm_PortType = 102
	// HuaweiIfm_PortType_50GE corresponds to the value 50GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_50GE E_HuaweiIfm_PortType = 103
	// HuaweiIfm_PortType_FlexE_50G corresponds to the value FlexE_50G of HuaweiIfm_PortType
	HuaweiIfm_PortType_FlexE_50G E_HuaweiIfm_PortType = 104
	// HuaweiIfm_PortType_FlexE_100G corresponds to the value FlexE_100G of HuaweiIfm_PortType
	HuaweiIfm_PortType_FlexE_100G E_HuaweiIfm_PortType = 105
	// HuaweiIfm_PortType_FlexE_50_OR_100G corresponds to the value FlexE_50|100G of HuaweiIfm_PortType
	HuaweiIfm_PortType_FlexE_50_OR_100G E_HuaweiIfm_PortType = 106
	// HuaweiIfm_PortType_Virtual_Serial corresponds to the value Virtual_Serial of HuaweiIfm_PortType
	HuaweiIfm_PortType_Virtual_Serial E_HuaweiIfm_PortType = 108
	// HuaweiIfm_PortType_400GE corresponds to the value 400GE of HuaweiIfm_PortType
	HuaweiIfm_PortType_400GE E_HuaweiIfm_PortType = 109
	// HuaweiIfm_PortType_HPGE corresponds to the value HPGE of HuaweiIfm_PortType
	HuaweiIfm_PortType_HPGE E_HuaweiIfm_PortType = 115
	// HuaweiIfm_PortType_FlexE_400G corresponds to the value FlexE_400G of HuaweiIfm_PortType
	HuaweiIfm_PortType_FlexE_400G E_HuaweiIfm_PortType = 116
	// HuaweiIfm_PortType_Virtual_if corresponds to the value Virtual_if of HuaweiIfm_PortType
	HuaweiIfm_PortType_Virtual_if E_HuaweiIfm_PortType = 117
	// HuaweiIfm_PortType_Cellular corresponds to the value Cellular of HuaweiIfm_PortType
	HuaweiIfm_PortType_Cellular E_HuaweiIfm_PortType = 118
)

// E_HuaweiIfm_RouterType is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_RouterType. An additional value named
// HuaweiIfm_RouterType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_RouterType int64

// IsYANGGoEnum ensures that HuaweiIfm_RouterType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_RouterType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_RouterType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_RouterType.
func (E_HuaweiIfm_RouterType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_RouterType.
func (e E_HuaweiIfm_RouterType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_RouterType")
}

const (
	// HuaweiIfm_RouterType_UNSET corresponds to the value UNSET of HuaweiIfm_RouterType
	HuaweiIfm_RouterType_UNSET E_HuaweiIfm_RouterType = 0
	// HuaweiIfm_RouterType_PtoP corresponds to the value PtoP of HuaweiIfm_RouterType
	HuaweiIfm_RouterType_PtoP E_HuaweiIfm_RouterType = 1
	// HuaweiIfm_RouterType_PtoMP corresponds to the value PtoMP of HuaweiIfm_RouterType
	HuaweiIfm_RouterType_PtoMP E_HuaweiIfm_RouterType = 2
	// HuaweiIfm_RouterType_broadcast corresponds to the value broadcast of HuaweiIfm_RouterType
	HuaweiIfm_RouterType_broadcast E_HuaweiIfm_RouterType = 3
	// HuaweiIfm_RouterType_NBMA corresponds to the value NBMA of HuaweiIfm_RouterType
	HuaweiIfm_RouterType_NBMA E_HuaweiIfm_RouterType = 4
	// HuaweiIfm_RouterType_invalid corresponds to the value invalid of HuaweiIfm_RouterType
	HuaweiIfm_RouterType_invalid E_HuaweiIfm_RouterType = 256
)

// E_HuaweiIfm_ServiceType is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_ServiceType. An additional value named
// HuaweiIfm_ServiceType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_ServiceType int64

// IsYANGGoEnum ensures that HuaweiIfm_ServiceType implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_ServiceType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_ServiceType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_ServiceType.
func (E_HuaweiIfm_ServiceType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_ServiceType.
func (e E_HuaweiIfm_ServiceType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_ServiceType")
}

const (
	// HuaweiIfm_ServiceType_UNSET corresponds to the value UNSET of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_UNSET E_HuaweiIfm_ServiceType = 0
	// HuaweiIfm_ServiceType_none corresponds to the value none of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_none E_HuaweiIfm_ServiceType = 1
	// HuaweiIfm_ServiceType_trunk_member corresponds to the value trunk_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_trunk_member E_HuaweiIfm_ServiceType = 3
	// HuaweiIfm_ServiceType_stack_member corresponds to the value stack_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_stack_member E_HuaweiIfm_ServiceType = 7
	// HuaweiIfm_ServiceType_mp_member corresponds to the value mp_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_mp_member E_HuaweiIfm_ServiceType = 8
	// HuaweiIfm_ServiceType_vbridge_member corresponds to the value vbridge_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_vbridge_member E_HuaweiIfm_ServiceType = 9
	// HuaweiIfm_ServiceType_ima_member corresponds to the value ima_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_ima_member E_HuaweiIfm_ServiceType = 10
	// HuaweiIfm_ServiceType_bundle_member corresponds to the value bundle_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_bundle_member E_HuaweiIfm_ServiceType = 11
	// HuaweiIfm_ServiceType_fabric_member corresponds to the value fabric_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_fabric_member E_HuaweiIfm_ServiceType = 12
	// HuaweiIfm_ServiceType_lag_master_member corresponds to the value lag_master_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_lag_master_member E_HuaweiIfm_ServiceType = 13
	// HuaweiIfm_ServiceType_lag_slave_member corresponds to the value lag_slave_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_lag_slave_member E_HuaweiIfm_ServiceType = 14
	// HuaweiIfm_ServiceType_cpos_trunk_member corresponds to the value cpos_trunk_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_cpos_trunk_member E_HuaweiIfm_ServiceType = 16
	// HuaweiIfm_ServiceType_pos_trunk_member corresponds to the value pos_trunk_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_pos_trunk_member E_HuaweiIfm_ServiceType = 17
	// HuaweiIfm_ServiceType_global_mp_member corresponds to the value global_mp_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_global_mp_member E_HuaweiIfm_ServiceType = 18
	// HuaweiIfm_ServiceType_global_ima_member corresponds to the value global_ima_member of HuaweiIfm_ServiceType
	HuaweiIfm_ServiceType_global_ima_member E_HuaweiIfm_ServiceType = 19
)

// E_HuaweiIfm_StatisticMode is a derived int64 type which is used to represent
// the enumerated node HuaweiIfm_StatisticMode. An additional value named
// HuaweiIfm_StatisticMode_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiIfm_StatisticMode int64

// IsYANGGoEnum ensures that HuaweiIfm_StatisticMode implements the yang.GoEnum
// interface. This ensures that HuaweiIfm_StatisticMode can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiIfm_StatisticMode) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiIfm_StatisticMode.
func (E_HuaweiIfm_StatisticMode) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiIfm_StatisticMode.
func (e E_HuaweiIfm_StatisticMode) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiIfm_StatisticMode")
}

const (
	// HuaweiIfm_StatisticMode_UNSET corresponds to the value UNSET of HuaweiIfm_StatisticMode
	HuaweiIfm_StatisticMode_UNSET E_HuaweiIfm_StatisticMode = 0
	// HuaweiIfm_StatisticMode_interface_based corresponds to the value interface_based of HuaweiIfm_StatisticMode
	HuaweiIfm_StatisticMode_interface_based E_HuaweiIfm_StatisticMode = 2
	// HuaweiIfm_StatisticMode_vlan_group_based corresponds to the value vlan_group_based of HuaweiIfm_StatisticMode
	HuaweiIfm_StatisticMode_vlan_group_based E_HuaweiIfm_StatisticMode = 3
)

// E_HuaweiSystem_RiskLevelType is a derived int64 type which is used to represent
// the enumerated node HuaweiSystem_RiskLevelType. An additional value named
// HuaweiSystem_RiskLevelType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiSystem_RiskLevelType int64

// IsYANGGoEnum ensures that HuaweiSystem_RiskLevelType implements the yang.GoEnum
// interface. This ensures that HuaweiSystem_RiskLevelType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiSystem_RiskLevelType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiSystem_RiskLevelType.
func (E_HuaweiSystem_RiskLevelType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiSystem_RiskLevelType.
func (e E_HuaweiSystem_RiskLevelType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiSystem_RiskLevelType")
}

const (
	// HuaweiSystem_RiskLevelType_UNSET corresponds to the value UNSET of HuaweiSystem_RiskLevelType
	HuaweiSystem_RiskLevelType_UNSET E_HuaweiSystem_RiskLevelType = 0
	// HuaweiSystem_RiskLevelType_high corresponds to the value high of HuaweiSystem_RiskLevelType
	HuaweiSystem_RiskLevelType_high E_HuaweiSystem_RiskLevelType = 1
	// HuaweiSystem_RiskLevelType_medium corresponds to the value medium of HuaweiSystem_RiskLevelType
	HuaweiSystem_RiskLevelType_medium E_HuaweiSystem_RiskLevelType = 2
	// HuaweiSystem_RiskLevelType_low corresponds to the value low of HuaweiSystem_RiskLevelType
	HuaweiSystem_RiskLevelType_low E_HuaweiSystem_RiskLevelType = 3
)

// E_HuaweiSystem_SecurityRiskType is a derived int64 type which is used to represent
// the enumerated node HuaweiSystem_SecurityRiskType. An additional value named
// HuaweiSystem_SecurityRiskType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiSystem_SecurityRiskType int64

// IsYANGGoEnum ensures that HuaweiSystem_SecurityRiskType implements the yang.GoEnum
// interface. This ensures that HuaweiSystem_SecurityRiskType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiSystem_SecurityRiskType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiSystem_SecurityRiskType.
func (E_HuaweiSystem_SecurityRiskType) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	return ΛEnum
}

// String returns a logging-friendly string for E_HuaweiSystem_SecurityRiskType.
func (e E_HuaweiSystem_SecurityRiskType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiSystem_SecurityRiskType")
}

const (
	// HuaweiSystem_SecurityRiskType_UNSET corresponds to the value UNSET of HuaweiSystem_SecurityRiskType
	HuaweiSystem_SecurityRiskType_UNSET E_HuaweiSystem_SecurityRiskType = 0
	// HuaweiSystem_SecurityRiskType_insecure_algorithm corresponds to the value insecure_algorithm of HuaweiSystem_SecurityRiskType
	HuaweiSystem_SecurityRiskType_insecure_algorithm E_HuaweiSystem_SecurityRiskType = 1
	// HuaweiSystem_SecurityRiskType_insecure_protocol corresponds to the value insecure_protocol of HuaweiSystem_SecurityRiskType
	HuaweiSystem_SecurityRiskType_insecure_protocol E_HuaweiSystem_SecurityRiskType = 2
	// HuaweiSystem_SecurityRiskType_insecure_configuration corresponds to the value insecure_configuration of HuaweiSystem_SecurityRiskType
	HuaweiSystem_SecurityRiskType_insecure_configuration E_HuaweiSystem_SecurityRiskType = 3
)

// E_HuaweiVlan_AccessType is a derived int64 type which is used to represent
// the enumerated node HuaweiVlan_AccessType. An additional value named
// HuaweiVlan_AccessType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiVlan_AccessType int64

// IsYANGGoEnum ensures that HuaweiVlan_AccessType implements the yang.GoEnum
// interface. This ensures that HuaweiVlan_AccessType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiVlan_AccessType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiVlan_AccessType.
func (E_HuaweiVlan_AccessType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiVlan_AccessType.
func (e E_HuaweiVlan_AccessType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiVlan_AccessType")
}

const (
	// HuaweiVlan_AccessType_UNSET corresponds to the value UNSET of HuaweiVlan_AccessType
	HuaweiVlan_AccessType_UNSET E_HuaweiVlan_AccessType = 0
	// HuaweiVlan_AccessType_access corresponds to the value access of HuaweiVlan_AccessType
	HuaweiVlan_AccessType_access E_HuaweiVlan_AccessType = 2
	// HuaweiVlan_AccessType_trunk corresponds to the value trunk of HuaweiVlan_AccessType
	HuaweiVlan_AccessType_trunk E_HuaweiVlan_AccessType = 3
	// HuaweiVlan_AccessType_hybrid corresponds to the value hybrid of HuaweiVlan_AccessType
	HuaweiVlan_AccessType_hybrid E_HuaweiVlan_AccessType = 4
	// HuaweiVlan_AccessType_dot1qtunnel corresponds to the value dot1qtunnel of HuaweiVlan_AccessType
	HuaweiVlan_AccessType_dot1qtunnel E_HuaweiVlan_AccessType = 5
)

// E_HuaweiVlan_AdminStatus is a derived int64 type which is used to represent
// the enumerated node HuaweiVlan_AdminStatus. An additional value named
// HuaweiVlan_AdminStatus_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiVlan_AdminStatus int64

// IsYANGGoEnum ensures that HuaweiVlan_AdminStatus implements the yang.GoEnum
// interface. This ensures that HuaweiVlan_AdminStatus can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiVlan_AdminStatus) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiVlan_AdminStatus.
func (E_HuaweiVlan_AdminStatus) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiVlan_AdminStatus.
func (e E_HuaweiVlan_AdminStatus) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiVlan_AdminStatus")
}

const (
	// HuaweiVlan_AdminStatus_UNSET corresponds to the value UNSET of HuaweiVlan_AdminStatus
	HuaweiVlan_AdminStatus_UNSET E_HuaweiVlan_AdminStatus = 0
	// HuaweiVlan_AdminStatus_down corresponds to the value down of HuaweiVlan_AdminStatus
	HuaweiVlan_AdminStatus_down E_HuaweiVlan_AdminStatus = 1
	// HuaweiVlan_AdminStatus_up corresponds to the value up of HuaweiVlan_AdminStatus
	HuaweiVlan_AdminStatus_up E_HuaweiVlan_AdminStatus = 2
)

// E_HuaweiVlan_EnableStatus is a derived int64 type which is used to represent
// the enumerated node HuaweiVlan_EnableStatus. An additional value named
// HuaweiVlan_EnableStatus_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiVlan_EnableStatus int64

// IsYANGGoEnum ensures that HuaweiVlan_EnableStatus implements the yang.GoEnum
// interface. This ensures that HuaweiVlan_EnableStatus can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiVlan_EnableStatus) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiVlan_EnableStatus.
func (E_HuaweiVlan_EnableStatus) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiVlan_EnableStatus.
func (e E_HuaweiVlan_EnableStatus) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiVlan_EnableStatus")
}

const (
	// HuaweiVlan_EnableStatus_UNSET corresponds to the value UNSET of HuaweiVlan_EnableStatus
	HuaweiVlan_EnableStatus_UNSET E_HuaweiVlan_EnableStatus = 0
	// HuaweiVlan_EnableStatus_disable corresponds to the value disable of HuaweiVlan_EnableStatus
	HuaweiVlan_EnableStatus_disable E_HuaweiVlan_EnableStatus = 1
	// HuaweiVlan_EnableStatus_enable corresponds to the value enable of HuaweiVlan_EnableStatus
	HuaweiVlan_EnableStatus_enable E_HuaweiVlan_EnableStatus = 2
)

// E_HuaweiVlan_OperStatus is a derived int64 type which is used to represent
// the enumerated node HuaweiVlan_OperStatus. An additional value named
// HuaweiVlan_OperStatus_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiVlan_OperStatus int64

// IsYANGGoEnum ensures that HuaweiVlan_OperStatus implements the yang.GoEnum
// interface. This ensures that HuaweiVlan_OperStatus can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiVlan_OperStatus) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiVlan_OperStatus.
func (E_HuaweiVlan_OperStatus) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiVlan_OperStatus.
func (e E_HuaweiVlan_OperStatus) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiVlan_OperStatus")
}

const (
	// HuaweiVlan_OperStatus_UNSET corresponds to the value UNSET of HuaweiVlan_OperStatus
	HuaweiVlan_OperStatus_UNSET E_HuaweiVlan_OperStatus = 0
	// HuaweiVlan_OperStatus_down corresponds to the value down of HuaweiVlan_OperStatus
	HuaweiVlan_OperStatus_down E_HuaweiVlan_OperStatus = 1
	// HuaweiVlan_OperStatus_up corresponds to the value up of HuaweiVlan_OperStatus
	HuaweiVlan_OperStatus_up E_HuaweiVlan_OperStatus = 2
)

// E_HuaweiVlan_TagMode is a derived int64 type which is used to represent
// the enumerated node HuaweiVlan_TagMode. An additional value named
// HuaweiVlan_TagMode_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiVlan_TagMode int64

// IsYANGGoEnum ensures that HuaweiVlan_TagMode implements the yang.GoEnum
// interface. This ensures that HuaweiVlan_TagMode can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiVlan_TagMode) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiVlan_TagMode.
func (E_HuaweiVlan_TagMode) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiVlan_TagMode.
func (e E_HuaweiVlan_TagMode) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiVlan_TagMode")
}

const (
	// HuaweiVlan_TagMode_UNSET corresponds to the value UNSET of HuaweiVlan_TagMode
	HuaweiVlan_TagMode_UNSET E_HuaweiVlan_TagMode = 0
	// HuaweiVlan_TagMode_untag corresponds to the value untag of HuaweiVlan_TagMode
	HuaweiVlan_TagMode_untag E_HuaweiVlan_TagMode = 1
	// HuaweiVlan_TagMode_tag corresponds to the value tag of HuaweiVlan_TagMode
	HuaweiVlan_TagMode_tag E_HuaweiVlan_TagMode = 2
)

// E_HuaweiVlan_VlanType is a derived int64 type which is used to represent
// the enumerated node HuaweiVlan_VlanType. An additional value named
// HuaweiVlan_VlanType_UNSET is added to the enumeration which is used as
// the nil value, indicating that the enumeration was not explicitly set by
// the program importing the generated structures.
type E_HuaweiVlan_VlanType int64

// IsYANGGoEnum ensures that HuaweiVlan_VlanType implements the yang.GoEnum
// interface. This ensures that HuaweiVlan_VlanType can be identified as a
// mapped type for a YANG enumeration.
func (E_HuaweiVlan_VlanType) IsYANGGoEnum() {}

// ΛMap returns the value lookup map associated with  HuaweiVlan_VlanType.
func (E_HuaweiVlan_VlanType) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

// String returns a logging-friendly string for E_HuaweiVlan_VlanType.
func (e E_HuaweiVlan_VlanType) String() string {
	return ygot.EnumLogString(e, int64(e), "E_HuaweiVlan_VlanType")
}

const (
	// HuaweiVlan_VlanType_UNSET corresponds to the value UNSET of HuaweiVlan_VlanType
	HuaweiVlan_VlanType_UNSET E_HuaweiVlan_VlanType = 0
	// HuaweiVlan_VlanType_common corresponds to the value common of HuaweiVlan_VlanType
	HuaweiVlan_VlanType_common E_HuaweiVlan_VlanType = 2
	// HuaweiVlan_VlanType_super corresponds to the value super of HuaweiVlan_VlanType
	HuaweiVlan_VlanType_super E_HuaweiVlan_VlanType = 3
	// HuaweiVlan_VlanType_sub corresponds to the value sub of HuaweiVlan_VlanType
	HuaweiVlan_VlanType_sub E_HuaweiVlan_VlanType = 4
)

// ΛEnum is a map, keyed by the name of the type defined for each enum in the
// generated Go code, which provides a mapping between the constant int64 value
// of each value of the enumeration, and the string that is used to represent it
// in the YANG schema. The map is named ΛEnum in order to avoid clash with any
// valid YANG identifier.
var ΛEnum = map[string]map[int64]ygot.EnumDefinition{
	"E_HuaweiIfm_ClassType": {
		1: {Name: "main-interface"},
		2: {Name: "sub-interface"},
	},
	"E_HuaweiIfm_DampLevelType": {
		1: {Name: "light"},
		2: {Name: "middle"},
		3: {Name: "heavy"},
	},
	"E_HuaweiIfm_DampStatusType": {
		1: {Name: "suppressed"},
		2: {Name: "unsuppressed"},
	},
	"E_HuaweiIfm_EncapsulationType": {
		1:   {Name: "vlan-type"},
		2:   {Name: "dot1q"},
		3:   {Name: "qinq"},
		4:   {Name: "p2p"},
		5:   {Name: "p2mp"},
		6:   {Name: "l2ve"},
		7:   {Name: "l3ve"},
		8:   {Name: "vlan-type-policy"},
		9:   {Name: "dot1q-policy"},
		10:  {Name: "stacking-policy"},
		11:  {Name: "untag-policy"},
		12:  {Name: "qinq-mapping"},
		13:  {Name: "l2vc"},
		14:  {Name: "l3vc"},
		15:  {Name: "evc-untag"},
		16:  {Name: "evc-dot1q"},
		17:  {Name: "evc-qinq"},
		18:  {Name: "evc-default"},
		19:  {Name: "evc-dot1q-policy"},
		20:  {Name: "ietf"},
		21:  {Name: "nonstandard"},
		22:  {Name: "user-vlan"},
		23:  {Name: "user-vlan-anyother"},
		24:  {Name: "qin-link"},
		25:  {Name: "soft-gre-ve"},
		26:  {Name: "l3ve-ter"},
		27:  {Name: "l3ve-acc"},
		256: {Name: "invalid"},
	},
	"E_HuaweiIfm_ErrorDownType": {
		1:  {Name: "bpdu-protection"},
		2:  {Name: "auto-defend"},
		3:  {Name: "monitor-link"},
		66: {Name: "portsec-reached-limit"},
		67: {Name: "storm-control"},
		68: {Name: "loopback-detect"},
		69: {Name: "dual-active"},
		70: {Name: "mac-address-flapping"},
		71: {Name: "no-stack-link"},
		72: {Name: "crc-statistics"},
		73: {Name: "transceiver-power-low"},
		74: {Name: "link-flap"},
		75: {Name: "l2-loop-occured"},
		76: {Name: "stack-member-exceed-limit"},
		77: {Name: "spine-member-exceed-limit"},
		78: {Name: "resource-mismatch"},
		79: {Name: "leaf-mstp"},
		80: {Name: "m-lag"},
		81: {Name: "fabric-uplink-threshold"},
		82: {Name: "stack-config-conflict"},
		83: {Name: "spine-type-unsupported"},
		84: {Name: "stack-packet-defensive"},
		86: {Name: "forward-engine-buffer-failed"},
		87: {Name: "forward-engine-interface-failed"},
		88: {Name: "fabric-link-failure"},
		89: {Name: "m-lag-consistency-check"},
		90: {Name: "pfc-deadlock"},
	},
	"E_HuaweiIfm_LinkProtocol": {
		1: {Name: "ethernet"},
		2: {Name: "ppp"},
		3: {Name: "hdlc"},
		4: {Name: "fr"},
		7: {Name: "atm"},
		8: {Name: "tdm"},
	},
	"E_HuaweiIfm_LinkQualityGradeType": {
		1: {Name: "good"},
		2: {Name: "high"},
		3: {Name: "middle"},
		5: {Name: "low"},
	},
	"E_HuaweiIfm_NetworkLayerState": {
		1:      {Name: "ipv4-ipv6-up"},
		196609: {Name: "ipv4-ipv6-down"},
	},
	"E_HuaweiIfm_PortStatus": {
		1: {Name: "down"},
		2: {Name: "up"},
	},
	"E_HuaweiIfm_PortType": {
		1:   {Name: "Ethernet"},
		3:   {Name: "GigabitEthernet"},
		5:   {Name: "Eth-Trunk"},
		6:   {Name: "Ip-Trunk"},
		7:   {Name: "Virtual-Ethernet"},
		9:   {Name: "Serial"},
		10:  {Name: "Pos"},
		11:  {Name: "Cpos"},
		12:  {Name: "ATM"},
		15:  {Name: "Tunnel"},
		16:  {Name: "Vlanif"},
		19:  {Name: "NULL"},
		20:  {Name: "LoopBack"},
		21:  {Name: "100GE"},
		22:  {Name: "Lmpif"},
		23:  {Name: "MTunnel"},
		24:  {Name: "40GE"},
		25:  {Name: "10GE"},
		26:  {Name: "GEBrief"},
		27:  {Name: "MEth"},
		28:  {Name: "Stack-Port"},
		29:  {Name: "Sip"},
		31:  {Name: "E1"},
		32:  {Name: "Mp-group"},
		33:  {Name: "Ima-group"},
		34:  {Name: "VMEth"},
		36:  {Name: "Remote-Ap"},
		37:  {Name: "VBridge"},
		38:  {Name: "Atm-Bundle"},
		39:  {Name: "Fiber-Channel"},
		40:  {Name: "Infiniband"},
		41:  {Name: "Vbdif"},
		42:  {Name: "T1"},
		43:  {Name: "T3"},
		44:  {Name: "VC4"},
		45:  {Name: "VC12"},
		46:  {Name: "Global-VE"},
		47:  {Name: "Fabric-Port"},
		49:  {Name: "E3"},
		50:  {Name: "Vp"},
		51:  {Name: "DcnInterface"},
		52:  {Name: "Cpos-Trunk"},
		53:  {Name: "Trunk-Serial"},
		54:  {Name: "Global-Mp-Group"},
		56:  {Name: "Otn"},
		58:  {Name: "Global-Ima-Group"},
		60:  {Name: "Pos-Trunk"},
		64:  {Name: "Gmpls-Uni"},
		65:  {Name: "Wdm"},
		66:  {Name: "Nve"},
		68:  {Name: "FCoE-Port"},
		69:  {Name: "Virtual-Template"},
		71:  {Name: "FC"},
		72:  {Name: "4x10GE"},
		73:  {Name: "10x10GE"},
		74:  {Name: "3x40GE"},
		75:  {Name: "4x25GE"},
		76:  {Name: "25GE"},
		80:  {Name: "IMEth"},
		89:  {Name: "PW-VE"},
		90:  {Name: "VX-Tunnel"},
		91:  {Name: "ServiceIf"},
		92:  {Name: "XGigabitEthernet"},
		93:  {Name: "200GE"},
		95:  {Name: "Virtual-ODUk"},
		96:  {Name: "FlexE"},
		97:  {Name: "FlexE-200GE"},
		102: {Name: "50|100GE"},
		103: {Name: "50GE"},
		104: {Name: "FlexE-50G"},
		105: {Name: "FlexE-100G"},
		106: {Name: "FlexE-50|100G"},
		108: {Name: "Virtual-Serial"},
		109: {Name: "400GE"},
		115: {Name: "HPGE"},
		116: {Name: "FlexE-400G"},
		117: {Name: "Virtual-if"},
		118: {Name: "Cellular"},
	},
	"E_HuaweiIfm_RouterType": {
		1:   {Name: "PtoP"},
		2:   {Name: "PtoMP"},
		3:   {Name: "broadcast"},
		4:   {Name: "NBMA"},
		256: {Name: "invalid"},
	},
	"E_HuaweiIfm_ServiceType": {
		1:  {Name: "none"},
		3:  {Name: "trunk-member"},
		7:  {Name: "stack-member"},
		8:  {Name: "mp-member"},
		9:  {Name: "vbridge-member"},
		10: {Name: "ima-member"},
		11: {Name: "bundle-member"},
		12: {Name: "fabric-member"},
		13: {Name: "lag-master-member"},
		14: {Name: "lag-slave-member"},
		16: {Name: "cpos-trunk-member"},
		17: {Name: "pos-trunk-member"},
		18: {Name: "global-mp-member"},
		19: {Name: "global-ima-member"},
	},
	"E_HuaweiIfm_StatisticMode": {
		2: {Name: "interface-based"},
		3: {Name: "vlan-group-based"},
	},
	"E_HuaweiSystem_RiskLevelType": {
		1: {Name: "high"},
		2: {Name: "medium"},
		3: {Name: "low"},
	},
	"E_HuaweiSystem_SecurityRiskType": {
		1: {Name: "insecure-algorithm"},
		2: {Name: "insecure-protocol"},
		3: {Name: "insecure-configuration"},
	},
	"E_HuaweiVlan_AccessType": {
		2: {Name: "access"},
		3: {Name: "trunk"},
		4: {Name: "hybrid"},
		5: {Name: "dot1qtunnel"},
	},
	"E_HuaweiVlan_AdminStatus": {
		1: {Name: "down"},
		2: {Name: "up"},
	},
	"E_HuaweiVlan_EnableStatus": {
		1: {Name: "disable"},
		2: {Name: "enable"},
	},
	"E_HuaweiVlan_OperStatus": {
		1: {Name: "down"},
		2: {Name: "up"},
	},
	"E_HuaweiVlan_TagMode": {
		1: {Name: "untag"},
		2: {Name: "tag"},
	},
	"E_HuaweiVlan_VlanType": {
		2: {Name: "common"},
		3: {Name: "super"},
		4: {Name: "sub"},
	},
}
