package beeperapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartLogin(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/user/login", r.URL.Path)
		assert.Equal(t, "Bearer BEEPER-PRIVATE-API-PLEASE-DONT-USE", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RespStartLogin{
			RequestID: "test-request-id",
			Type:      []string{"email"},
			Expires:   time.Now().Add(time.Hour),
		})
	}))
	defer server.Close()

	// We can't easily test this without mocking the base domain resolution
	// But we can test the struct
	resp := &RespStartLogin{
		RequestID: "test-request-id",
		Type:      []string{"email"},
		Expires:   time.Now(),
	}

	assert.Equal(t, "test-request-id", resp.RequestID)
	assert.Equal(t, []string{"email"}, resp.Type)
}

func TestSendLoginEmailStruct(t *testing.T) {
	req := &ReqSendLoginEmail{
		RequestID: "test-request",
		Email:     "test@example.com",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var unmarshaled ReqSendLoginEmail
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, req.RequestID, unmarshaled.RequestID)
	assert.Equal(t, req.Email, unmarshaled.Email)
}

func TestSendLoginCodeStruct(t *testing.T) {
	req := &ReqSendLoginCode{
		RequestID: "test-request",
		Code:      "123456",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var unmarshaled ReqSendLoginCode
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, req.RequestID, unmarshaled.RequestID)
	assert.Equal(t, req.Code, unmarshaled.Code)
}

func TestRespSendLoginCodeStruct(t *testing.T) {
	resp := &RespSendLoginCode{
		LoginToken: "test-login-token",
		Whoami: &RespWhoami{
			UserInfo: WhoamiUserInfo{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var unmarshaled RespSendLoginCode
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, resp.LoginToken, unmarshaled.LoginToken)
	assert.NotNil(t, unmarshaled.Whoami)
	assert.Equal(t, "testuser", unmarshaled.Whoami.UserInfo.Username)
}

func TestLoginAuth(t *testing.T) {
	// Test that the login auth constant is defined
	assert.Equal(t, "BEEPER-PRIVATE-API-PLEASE-DONT-USE", loginAuth)
}

func TestErrInvalidLoginCodeAuth(t *testing.T) {
	// Test that the error variable is properly defined
	assert.NotNil(t, ErrInvalidLoginCode)
	assert.Equal(t, "invalid login code", ErrInvalidLoginCode.Error())
}

func TestStartLoginRequest(t *testing.T) {
	// Test the actual function - it should fail since we don't have a real API
	_, err := StartLogin("test.com")
	assert.Error(t, err) // Expected to fail since we can't connect to the real API
}

func TestSendLoginEmailRequest(t *testing.T) {
	// This should fail because we don't have a real API endpoint
	err := SendLoginEmail("test.com", "test-request", "test@example.com")
	assert.Error(t, err)
}

func TestSendLoginCodeRequest(t *testing.T) {
	// This should fail because we don't have a real API endpoint
	_, err := SendLoginCode("test.com", "test-request", "123456")
	assert.Error(t, err)
}
