package main

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/mediocregopher/radix/v3"
	"github.com/stretchr/testify/assert"

	"encoding/json"
	"net/http"
	"net/http/httptest"

	"strings"
	"testing"
)

const (
	// simulating a full Redis would be very complex. Let's use local Redis with high DB number
	redisTestUri = "redis://localhost:6379/15"
)

func setup(t *testing.T) *radix.Pool {
	redis, err := radix.NewPool("tcp", redisTestUri, 1)
	if err != nil {
		t.Fatal("Cannot connect to Redis on", redisTestUri)
	}
	return redis
}

// simple test for the redis access functions
func TestRedisFunctions(t *testing.T) {
	redis := setup(t)
	defer redis.Close()

	data := secretData{PubKey: "key", Nonce: "nonce", Secret: "secret"}
	mnemo, success := saveInRedis(redis, &data)
	if !success {
		t.Fatal("Failed to save value in Redis")
	}

	retrieved, success := loadFromRedis(redis, mnemo)
	if !success {
		t.Fatal("Failed to load data from Redis")
	}

	assert.Equal(t, retrieved.PubKey, "key")
	assert.Equal(t, retrieved.Nonce, "nonce")
	assert.Equal(t, retrieved.Secret, "secret")

	// check that the value is now gone from redis
	var value string
	err := redis.Do(radix.Cmd(&value, "GET", mnemo))
	if err != nil {
		t.Fatal("Failed to check value in Redis")
	}
	assert.Len(t, value, 0)
}

func TestStaticRoutes(t *testing.T) {
	redis := setup(t)
	defer redis.Close()
	router := setupRouter(redis)
	recorder := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)

	req, _ = http.NewRequest("GET", "/imprint", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)

	req, _ = http.NewRequest("GET", "/r", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)

	req, _ = http.NewRequest("GET", "/assets/keedrop.js", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)
}

// Simple round-trip integration test
func TestApiRequests(t *testing.T) {
	redis := setup(t)
	defer redis.Close()
	router := setupRouter(redis)
	recorder := httptest.NewRecorder()
	const postBody = "{\"pubkey\":\"fT8w5J5ByGwwZ!Ew8lAEaf4x+92m93iGvV1PA3KMewk=\",\"nonce\":\"NerymzRamnXBjZuk2UFqg2hOmhwJQfUx\",\"secret\":\"8A0mQy49FMxk7p2UN3Q5nxGA373xgN5B4g==\"}"
	req, _ := http.NewRequest("POST", "/api/secret", strings.NewReader(postBody))
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)

	var responseData struct {
		Mnemo string `json:"mnemo" binding:"required"`
	}
	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &responseData))

	req, _ = http.NewRequest("GET", "/api/secret/"+responseData.Mnemo, nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)
}

func TestCorsPreflightRequests(t *testing.T) {
	redis := setup(t)
	defer redis.Close()
	router := setupRouter(redis)
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/secret", nil)
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Origin", "https://keedrop.de")
	router.ServeHTTP(recorder, req)
	// no env var set: the API is same-origin only, so no CORS headers
	assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Methods"))

	t.Setenv("KEEDROP_CORS_ORIGINS", "https://keedrop.de")
	router = setupRouter(redis)
	req.Header.Set("Origin", "https://fakedomain.com")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 403, recorder.Code)

	req.Header.Set("Origin", "https://keedrop.de")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 204, recorder.Code)
	assert.Equal(t, "POST,GET", recorder.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type", recorder.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "https://keedrop.de", recorder.Header().Get("Access-Control-Allow-Origin"))
}

func postSecret(router *gin.Engine, remoteAddr string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/secret", strings.NewReader(body))
	req.RemoteAddr = remoteAddr
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestOversizedSecretIsRejected(t *testing.T) {
	redis := setup(t)
	defer redis.Close()
	router := setupRouter(redis)

	hugeSecret := strings.Repeat("A", maxRequestBodyBytes)
	body := "{\"pubkey\":\"key\",\"nonce\":\"nonce\",\"secret\":\"" + hugeSecret + "\"}"
	recorder := postSecret(router, "192.0.2.1:1234", body)
	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "mnemo")
}

func TestApiIsRateLimitedPerClient(t *testing.T) {
	t.Setenv("KEEDROP_RATE_LIMIT", "3")
	redis := setup(t)
	defer redis.Close()
	router := setupRouter(redis)
	const body = "{\"pubkey\":\"key\",\"nonce\":\"nonce\",\"secret\":\"secret\"}"

	for i := 0; i < 3; i++ {
		assert.Equal(t, 200, postSecret(router, "192.0.2.1:1234", body).Code)
	}
	assert.Equal(t, http.StatusTooManyRequests, postSecret(router, "192.0.2.1:5678", body).Code)

	// a spoofed X-Forwarded-For from an untrusted client must not reset the limit
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/secret", strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)

	// other clients are not affected
	assert.Equal(t, 200, postSecret(router, "192.0.2.2:1234", body).Code)
}

func TestRequestLogDoesNotContainMnemo(t *testing.T) {
	var logged bytes.Buffer
	previous := gin.DefaultWriter
	gin.DefaultWriter = &logged
	defer func() { gin.DefaultWriter = previous }()

	redis := setup(t)
	defer redis.Close()
	router := setupRouter(redis)

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/secret/Zq7xWv3KpL", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 404, recorder.Code)
	assert.Contains(t, logged.String(), "/api/secret/:mnemo")
	assert.NotContains(t, logged.String(), "Zq7xWv3KpL")
}
