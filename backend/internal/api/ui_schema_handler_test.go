package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/stretchr/testify/assert"
)

type fakeUISchemaClient struct {
	data interface{}
}

func (f fakeUISchemaClient) Get(ctx context.Context, path string, opts ...client.GetOption) (*client.GetResult, error) {
	return &client.GetResult{Path: path, Data: f.data, Timestamp: time.Now()}, nil
}

func (f fakeUISchemaClient) Set(ctx context.Context, changes []client.Change, opts ...client.SetOption) (*client.SetResult, error) {
	return &client.SetResult{Success: true}, nil
}

func (f fakeUISchemaClient) Subscribe(ctx context.Context, path string, handler func(client.Notification)) error {
	return nil
}

func (f fakeUISchemaClient) Close() error {
	return nil
}

func (f fakeUISchemaClient) IsConnected() bool {
	return true
}

func (f fakeUISchemaClient) DiscardCandidate(ctx context.Context) error {
	return nil
}

func TestUISchemaHandlerGetInterfaces(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	handler := NewUISchemaHandler(nil)
	router.GET("/api/v1/ui-schema/devices/:ip/interfaces", handler.GetInterfaces)

	// Test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ui-schema/devices/192.168.1.1/interfaces", nil)
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, 0, response.Code)
}

func TestUISchemaHandlerGetInterfacesPopulatesDeviceValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	ifm := &huawei.HuaweiIfm_Ifm_Interfaces{}
	iface, err := ifm.NewInterface("GigabitEthernet0/0/1")
	assert.NoError(t, err)
	desc := "uplink"
	mtu := uint32(1500)
	iface.Description = &desc
	iface.Mtu = &mtu
	iface.AdminStatus = huawei.E_HuaweiIfm_PortStatus(2)

	m := manager.New(manager.WithClientFactory(func(info client.DeviceConnectionInfo) (client.Client, error) {
		return fakeUISchemaClient{data: ifm}, nil
	}))
	handler := NewUISchemaHandler(m)
	router.GET("/api/v1/ui-schema/devices/:ip/interfaces", handler.GetInterfaces)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ui-schema/devices/192.168.1.1/interfaces", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["success"])

	data := response["data"].(map[string]interface{})
	values := data["values"].(map[string]interface{})
	rows := values["interfaces-table"].([]interface{})
	assert.Len(t, rows, 1)
	row := rows[0].(map[string]interface{})
	assert.Equal(t, "GigabitEthernet0/0/1", row["name"])
	assert.Equal(t, "uplink", row["description"])
	assert.Equal(t, float64(1500), row["mtu"])
	assert.Equal(t, float64(2), row["admin-status"])
}

func TestUISchemaHandlerApplyRejectsInvalidMTU(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	handler := NewUISchemaHandler(nil)
	router.POST("/api/v1/ui-schema/devices/:ip/interfaces/apply", handler.ApplyInterfaces)

	// Test request with invalid MTU (42 is below 1280 minimum)
	requestBody := map[string]interface{}{
		"schemaVersion": "interfaces:v1",
		"values": map[string]interface{}{
			"interfaces-table": []interface{}{
				map[string]interface{}{
					"name": "GigabitEthernet0/0/1",
					"mtu":  42,
				},
			},
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ui-schema/devices/192.168.1.1/interfaces/apply", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	// Use a map to assert the string code
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Equal(t, "VALIDATION_FAILED", response["code"])
}

func TestUISchemaHandlerApplyPreservesSchemaVersionMismatchCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	handler := NewUISchemaHandler(nil)
	router.POST("/api/v1/ui-schema/devices/:ip/interfaces/apply", handler.ApplyInterfaces)

	requestBody := map[string]interface{}{
		"schemaVersion": "interfaces:old",
		"values": map[string]interface{}{
			"interfaces-table": []interface{}{},
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ui-schema/devices/192.168.1.1/interfaces/apply", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Equal(t, "SCHEMA_VERSION_MISMATCH", response["code"])
}
