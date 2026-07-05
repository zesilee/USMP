package manager

import (
	"context"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
	"github.com/stretchr/testify/assert"
)

// recorderCapturingController implements Controller + status.RecorderSetter.
type recorderCapturingController struct {
	name string
	got  status.Recorder
}

func (f *recorderCapturingController) Start(ctx context.Context) error     { return nil }
func (f *recorderCapturingController) Stop() error                         { return nil }
func (f *recorderCapturingController) Enqueue(evt predicate.Event)         {}
func (f *recorderCapturingController) Name() string                        { return f.name }
func (f *recorderCapturingController) SetStatusRecorder(r status.Recorder) { f.got = r }

// plainController implements Controller but NOT RecorderSetter.
type plainController struct{ name string }

func (p *plainController) Start(ctx context.Context) error { return nil }
func (p *plainController) Stop() error                     { return nil }
func (p *plainController) Enqueue(evt predicate.Event)     {}
func (p *plainController) Name() string                    { return p.name }

func TestManager_HasReconcileStatusStore(t *testing.T) {
	m := New()
	assert.NotNil(t, m.GetReconcileStatus())
}

func TestManager_AddController_InjectsSharedStore(t *testing.T) {
	m := New()
	c := &recorderCapturingController{name: "vlan"}
	m.AddController(c)

	if assert.NotNil(t, c.got, "AddController should inject a recorder into RecorderSetter controllers") {
		// Prove it is the SAME store the manager exposes: record via the injected
		// recorder, then read it back through the manager's getter.
		c.got.Record("10.0.0.1", "/vlans", status.OutcomeConverged, 0, nil)
		st, ok := m.GetReconcileStatus().Get("10.0.0.1", "/vlans")
		assert.True(t, ok)
		assert.Equal(t, status.OutcomeConverged, st.Outcome)
	}
}

func TestManager_AddController_PlainControllerOK(t *testing.T) {
	m := New()
	assert.NotPanics(t, func() { m.AddController(&plainController{name: "x"}) })
}

// Manager interface must expose GetReconcileStatus.
func TestManager_InterfaceExposesReconcileStatus(t *testing.T) {
	var m Manager = New()
	assert.NotNil(t, m.GetReconcileStatus())
}
