package api

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// TestNestedSchemaCarriesStringLengthFromRealVLAN（D9 债务清偿）：schema 端点把
// 真实 huawei-vlan 的字符串 `length` 约束透出到 FieldDef（minLength/maxLength），
// 使一期 FE-22「合法长度」占位自动生效。数据驱动（真实 YANG），非硬编码。
//   - vlan `name` 叶：length "1..31"
//   - vlan `description` 叶：length "1..80"
//   - 数值叶（无 length 语义）不携带长度字段
func TestNestedSchemaCarriesStringLengthFromRealVLAN(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("vlan")
	if !ok {
		t.Fatal("vlan module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	vlans, ok := findFieldBySuffix(ys.Fields, "vlans")
	if !ok {
		t.Fatal("vlans field not found")
	}
	list := vlans.Fields[0] // vlan list
	var name, desc, id *FieldDef
	for i := range list.Fields {
		switch list.Fields[i].Label {
		case "name":
			name = &list.Fields[i]
		case "description":
			desc = &list.Fields[i]
		case "id":
			id = &list.Fields[i]
		}
	}
	if name == nil || desc == nil || id == nil {
		t.Fatalf("vlan leaves not found (name=%v desc=%v id=%v)", name != nil, desc != nil, id != nil)
	}
	if name.MinLength != 1 || name.MaxLength != 31 {
		t.Errorf("name length = [%d,%d], want [1,31]", name.MinLength, name.MaxLength)
	}
	if desc.MinLength != 1 || desc.MaxLength != 80 {
		t.Errorf("description length = [%d,%d], want [1,80]", desc.MinLength, desc.MaxLength)
	}
	if id.MinLength != 0 || id.MaxLength != 0 {
		t.Errorf("numeric id must not carry length, got [%d,%d]", id.MinLength, id.MaxLength)
	}
}
