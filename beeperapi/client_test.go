package beeperapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	req := newRequest("test.com", "test-token", "GET", "/test")
	assert.NotNil(t, req)
	assert.Equal(t, "GET", req.Method)
	assert.Contains(t, req.URL.String(), "test.com")
	assert.Contains(t, req.URL.String(), "/test")
	assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
}

func TestEncodeContent(t *testing.T) {
	req := newRequest("test.com", "test-token", "POST", "/test")

	testData := map[string]string{
		"key": "value",
	}

	err := encodeContent(req, testData)
	assert.NoError(t, err)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

func TestEncodeContent_Nil(t *testing.T) {
	req := newRequest("test.com", "test-token", "GET", "/test")

	err := encodeContent(req, nil)
	assert.NoError(t, err)
}

func TestDoRequest_InvalidURL(t *testing.T) {
	req := newRequest("invalid-domain", "test-token", "GET", "/test")

	err := doRequest(req, nil, nil)
	assert.Error(t, err)
}

func TestDeleteBridge(t *testing.T) {
	// This should fail since we don't have a real API
	err := DeleteBridge("test.com", "test-bridge", "test-token")
	assert.Error(t, err)
}

func TestPostBridgeState(t *testing.T) {
	data := ReqPostBridgeState{
		StateEvent: "CONNECTED",
		Reason:     "test",
		Info:       map[string]any{"test": "value"},
	}

	// This should fail since we don't have a real API
	err := PostBridgeState("test.com", "testuser", "testbridge", "test-token", data)
	assert.Error(t, err)
}

func TestWhoami(t *testing.T) {
	// This should fail since we don't have a real API
	_, err := Whoami("test.com", "test-token")
	assert.Error(t, err)
}

func TestGetMatrixTokenFromJWT(t *testing.T) {
	// This should fail with an invalid JWT
	_, err := GetMatrixTokenFromJWT("invalid-jwt")
	assert.Error(t, err)
}

func TestGetMatrixTokenFromJWT_EmptyToken(t *testing.T) {
	// This should fail with an empty token
	_, err := GetMatrixTokenFromJWT("")
	assert.Error(t, err)
}
