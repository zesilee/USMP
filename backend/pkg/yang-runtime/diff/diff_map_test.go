package diff

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ygot 把 YANG list 生成为 map[key]*Entry（而非 slice），叶子为指针、枚举为值类型。
// 这些测试固定「合并/子集」语义：desired 是 UI 累积的意图子集，不是设备全树。
//   - desired 已设字段必须在 actual 上匹配，否则算漂移（需下发）
//   - desired 未设字段（零值：nil 指针 / 枚举 0 / "" / 0）不参与比较（不管理）
//   - actual 独有的 key（如物理口）忽略，绝不产生 DeleteChange
// 这样在设备落盘 desired 后能收敛（0 changes），修掉「新建接口后一直漂移」。

type mapIfmContainer struct {
	Interface map[string]*mapIfmEntry
}

type mapIfmEntry struct {
	Name        *string
	Description *string
	Mtu         *uint32
	AdminStatus int64 // ygot 枚举是值类型，0 = UNSET
	Members     map[string]*mapMember
}

type mapMember struct {
	Port *string
}

func sp(s string) *string  { return &s }
func u32(v uint32) *uint32 { return &v }

// 幂等/收敛：desired 稀疏条目 vs actual 同 key 全量条目（set 字段一致）→ 0 changes。
func TestDiffMap_SubsetMatch_Converges(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Description: sp("uplink")},
	}}
	actual := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		// 设备回读：同 key + 额外只读/默认字段（Mtu/AdminStatus）都填了
		"eth0": {Name: sp("eth0"), Description: sp("uplink"), Mtu: u32(1500), AdminStatus: 2},
	}}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Summary.Total, "desired 子集已被 actual 满足，应收敛")
}

// 新建接口：desired 有 key 而 actual 没有 → 需下发（1 change，NewValue 为整张 desired map）。
func TestDiffMap_NewKey_ProducesChange(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth9": {Name: sp("eth9"), Description: sp("new")},
	}}
	actual := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0")},
	}}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	require.Equal(t, 1, result.Summary.Total)
	ch := result.Changes[0]
	assert.Equal(t, ModifyChange, ch.Type)
	assert.Equal(t, "Interface", ch.Path)
	// NewValue 必须是 desired 的内层 map（供 marshalChange 走 IFM builder 整表下发）
	m, ok := ch.NewValue.(map[string]*mapIfmEntry)
	require.True(t, ok, "NewValue 应为 desired 内层 map, got %T", ch.NewValue)
	assert.Contains(t, m, "eth9")
}

// 不误删物理口：actual 有 desired 没有的 key，且 desired 子集匹配 → 0 changes（忽略不删）。
func TestDiffMap_IgnoresActualOnlyKeys(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Description: sp("uplink")},
	}}
	actual := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0":  {Name: sp("eth0"), Description: sp("uplink")},
		"phys1": {Name: sp("phys1")}, // 设备物理口，UI 未管理
		"phys2": {Name: sp("phys2")},
	}}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Summary.Total)
	assert.Zero(t, result.Summary.Deletes, "actual 独有的物理口绝不能被删除")
}

// 真实漂移：desired 的已设字段与 actual 不一致 → 1 change。
func TestDiffMap_LeafDrift_ProducesChange(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Description: sp("NEW")},
	}}
	actual := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Description: sp("OLD"), Mtu: u32(1500)},
	}}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
}

// 未设字段不驱动漂移：desired 只设 Description，actual 有一堆额外字段 → 收敛。
func TestDiffMap_UnsetDesiredFieldsIgnored(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Description: sp("uplink")}, // Name/Mtu/AdminStatus 未设
	}}
	actual := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Description: sp("uplink"), Mtu: u32(9000), AdminStatus: 2},
	}}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Summary.Total)
}

// 嵌套容器（成员口 map）子集匹配收敛。
func TestDiffMap_NestedMapSubsetMatch(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Members: map[string]*mapMember{"GE1": {Port: sp("GE1")}}},
	}}
	actual := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Members: map[string]*mapMember{
			"GE1": {Port: sp("GE1")},
			"GE2": {Port: sp("GE2")}, // 设备侧多出的成员，忽略
		}},
	}}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Summary.Total)
}

// 嵌套容器成员缺失 → 漂移。
func TestDiffMap_NestedMapMissingMember_Drift(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Members: map[string]*mapMember{"GE9": {Port: sp("GE9")}}},
	}}
	actual := &mapIfmContainer{Interface: map[string]*mapIfmEntry{
		"eth0": {Name: sp("eth0"), Members: map[string]*mapMember{"GE1": {Port: sp("GE1")}}},
	}}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
}

// 边界：空 desired map（无意图）vs 非空 actual → 无需下发。
func TestDiffMap_EmptyDesired_NoChange(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{}}
	actual := &mapIfmContainer{Interface: map[string]*mapIfmEntry{"eth0": {Name: sp("eth0")}}}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Summary.Total)
}

// 边界：actual map 为 nil（设备空配）但 desired 有条目 → 需下发。
func TestDiffMap_NilActualMap_ProducesChange(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &mapIfmContainer{Interface: map[string]*mapIfmEntry{"eth0": {Name: sp("eth0")}}}
	actual := &mapIfmContainer{Interface: nil}

	result, err := de.Diff(desired, actual, schema.NewSchema())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
}
