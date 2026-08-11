package object

import (
	"encoding/json"
	"testing"
)

func TestJSONArray(t *testing.T) {
	if got := string(JSONArray(nil)); got != "[]" {
		t.Fatalf("empty = %s", got)
	}
	got := string(JSONArray([]json.RawMessage{json.RawMessage(`1`), json.RawMessage(`"a"`)}))
	if got != `[1,"a"]` {
		t.Fatalf("join = %s", got)
	}
}

func TestStripModule(t *testing.T) {
	for in, want := range map[string]string{
		"huawei-vlan:vlan": "vlan", "vlan": "vlan", "a:b:c": "b:c",
	} {
		if got := StripModule(in); got != want {
			t.Errorf("StripModule(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParse64JSON(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want uint64
		ok   bool
	}{
		{`"18446744073709551615"`, 18446744073709551615, true},
		{`42`, 42, true},
		{`"42"`, 42, true},
		{`"abc"`, 0, false},
		{`-1`, 0, false},
	} {
		got, err := ParseUint64JSON(json.RawMessage(tc.raw))
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ParseUint64JSON(%s) = %d, %v", tc.raw, got, err)
		}
	}
	if v, err := ParseInt64JSON(json.RawMessage(`"-9223372036854775808"`)); err != nil || v != -9223372036854775808 {
		t.Errorf("ParseInt64JSON min = %d, %v", v, err)
	}
	if _, err := ParseInt64JSON(json.RawMessage(`"x"`)); err == nil {
		t.Error("ParseInt64JSON must reject non-numeric")
	}
}

func TestEnumValueByName(t *testing.T) {
	if v, ok := EnumValueByName(sampleEnumMaps, "E_Sample_AdminStatus", "up"); !ok || v != 2 {
		t.Fatalf("up = %d, %v", v, ok)
	}
	if _, ok := EnumValueByName(sampleEnumMaps, "E_Sample_AdminStatus", "nope"); ok {
		t.Fatal("unknown name must miss")
	}
	if _, ok := EnumValueByName(sampleEnumMaps, "E_Nope", "up"); ok {
		t.Fatal("unknown type must miss")
	}
}

func TestEmptyJSON(t *testing.T) {
	if !IsEmptyJSON(EmptyJSON) || !IsEmptyJSON(json.RawMessage(" [ null ] ")) {
		t.Fatal("canonical [null] must be recognized")
	}
	for _, bad := range []string{`null`, `[]`, `[null,null]`, `[0]`, `true`} {
		if IsEmptyJSON(json.RawMessage(bad)) {
			t.Fatalf("%s wrongly accepted as empty", bad)
		}
	}
}

func TestRawJSON(t *testing.T) {
	if string(RawJSON("a")) != `"a"` || string(RawJSON(uint16(7))) != "7" || string(RawJSON(true)) != "true" {
		t.Fatal("scalar RawJSON mismatch")
	}
}
