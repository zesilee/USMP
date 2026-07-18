package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlways(t *testing.T) {
	p := Always()
	assert.True(t, p.Create(Event{}))
	assert.True(t, p.Update(Event{}))
	assert.True(t, p.Delete(Event{}))
	assert.True(t, p.Generic(Event{}))
}

func TestNever(t *testing.T) {
	p := Never()
	assert.False(t, p.Create(Event{}))
	assert.False(t, p.Update(Event{}))
	assert.False(t, p.Delete(Event{}))
	assert.False(t, p.Generic(Event{}))
}

func TestAnd(t *testing.T) {
	p1 := Prefix("/interfaces")
	p2 := Exact("/interfaces/interface")
	p := And(p1, p2)

	evt := Event{Path: "/interfaces/interface"}
	assert.True(t, p.Create(evt))

	evt2 := Event{Path: "/interfaces/interface[name='eth0']"}
	assert.False(t, p2.Create(evt2))
	assert.False(t, p.Create(evt2))
}

func TestOr(t *testing.T) {
	p1 := Exact("/interfaces/interface")
	p2 := Exact("/system/hostname")
	p := Or(p1, p2)

	evt1 := Event{Path: "/interfaces/interface"}
	assert.True(t, p.Create(evt1))

	evt2 := Event{Path: "/system/hostname"}
	assert.True(t, p.Create(evt2))

	evt3 := Event{Path: "/system/ntp"}
	assert.False(t, p.Create(evt3))
}

func TestNot(t *testing.T) {
	p := Not(Prefix("/interfaces"))

	evt1 := Event{Path: "/system/hostname"}
	assert.True(t, p.Create(evt1))

	evt2 := Event{Path: "/interfaces/interface"}
	assert.False(t, p.Create(evt2))
}

func TestPrefix(t *testing.T) {
	p := Prefix("/interfaces")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "exact prefix match",
			path: "/interfaces",
			want: true,
		},
		{
			name: "child path",
			path: "/interfaces/interface",
			want: true,
		},
		{
			name: "different prefix",
			path: "/system/hostname",
			want: false,
		},
		{
			name: "partial match",
			path: "/interface",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := Event{Path: tt.path}
			assert.Equal(t, tt.want, p.Create(evt))
			assert.Equal(t, tt.want, p.Update(evt))
			assert.Equal(t, tt.want, p.Delete(evt))
			assert.Equal(t, tt.want, p.Generic(evt))
		})
	}
}

func TestExact(t *testing.T) {
	p := Exact("/interfaces/interface[name='eth0']/description")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "exact match",
			path: "/interfaces/interface[name='eth0']/description",
			want: true,
		},
		{
			name: "different path",
			path: "/interfaces/interface[name='eth0']/enabled",
			want: false,
		},
		{
			name: "parent path",
			path: "/interfaces/interface[name='eth0']",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := Event{Path: tt.path}
			assert.Equal(t, tt.want, p.Create(evt))
		})
	}
}

func TestContains(t *testing.T) {
	p := Contains("interface")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "contains substring",
			path: "/interfaces/interface",
			want: true,
		},
		{
			name: "does not contain",
			path: "/system/hostname",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := Event{Path: tt.path}
			assert.Equal(t, tt.want, p.Create(evt))
		})
	}
}

func TestModulePrefix(t *testing.T) {
	p := ModulePrefix("example-interfaces")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "correct module prefix",
			path: "/example-interfaces:interfaces/example-interfaces:interface",
			want: true,
		},
		{
			name: "wrong module",
			path: "/example-system:system",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := Event{Path: tt.path}
			assert.Equal(t, tt.want, p.Create(evt))
		})
	}
}

func TestDeviceIDEqual(t *testing.T) {
	p := DeviceIDEqual("192.168.1.1")

	evt1 := Event{DeviceID: "192.168.1.1"}
	assert.True(t, p.Create(evt1))

	evt2 := Event{DeviceID: "192.168.1.2"}
	assert.False(t, p.Create(evt2))
}

func TestByType(t *testing.T) {
	p := ByType(CreateEvent, UpdateEvent)

	tests := []struct {
		name string
		typ  EventType
		want bool
	}{
		{
			name: "create event accepted",
			typ:  CreateEvent,
			want: true,
		},
		{
			name: "update event accepted",
			typ:  UpdateEvent,
			want: true,
		},
		{
			name: "delete event rejected",
			typ:  DeleteEvent,
			want: false,
		},
		{
			name: "generic event rejected",
			typ:  GenericEvent,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := Event{Type: tt.typ}
			assert.Equal(t, tt.want, p.Create(evt))
		})
	}
}

func TestForRequest(t *testing.T) {
	evt := Event{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface",
		Type:     UpdateEvent,
	}

	req := evt.ForRequest()
	assert.Equal(t, "192.168.1.1", req.DeviceID)
	assert.Equal(t, "/interfaces/interface", req.Path)
}

func TestFuncsPartial(t *testing.T) {
	// When only some functions are set, others default to true
	p := Funcs{
		CreateFunc: func(evt Event) bool {
			return false
		},
	}

	evt := Event{}
	assert.False(t, p.Create(evt))
	assert.True(t, p.Update(evt))
	assert.True(t, p.Delete(evt))
	assert.True(t, p.Generic(evt))
}
