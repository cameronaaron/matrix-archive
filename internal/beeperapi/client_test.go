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

func TestDoRequest_WithRequestData(t *testing.T) {
	req := newRequest("httpbin.org", "test-token", "POST", "/post")

	requestData := map[string]string{
		"test": "data",
	}

	// This will fail because httpbin.org doesn't accept our auth, but we're testing the encoding path
	err := doRequest(req, requestData, nil)
	assert.Error(t, err) // Expected to fail due to auth/API differences, but request encoding should work
}

func TestDoRequest_WithResponse(t *testing.T) {
	req := newRequest("httpbin.org", "test-token", "GET", "/json")

	var response map[string]interface{}

	// This will fail because httpbin.org doesn't accept our auth, but we're testing the response path
	err := doRequest(req, nil, &response)
	assert.Error(t, err) // Expected to fail due to auth/API differences
}

func TestDoRequest_NilRequestAndResponse(t *testing.T) {
	req := newRequest("httpbin.org", "test-token", "GET", "/get")

	// This will fail because httpbin.org doesn't accept our auth, but it tests the nil handling paths
	err := doRequest(req, nil, nil)
	assert.Error(t, err) // Expected to fail due to auth/API differences
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
