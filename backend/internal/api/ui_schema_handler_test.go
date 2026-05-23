package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

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

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
}
