package main

import (
	"github.com/mediocregopher/radix/v3"
	"github.com/stretchr/testify/assert"
	"os"

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
	assert.Equal(t, 204, recorder.Code)
	assert.Equal(t, "POST,GET", recorder.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type", recorder.Header().Get("Access-Control-Allow-Headers"))
	// no env var set, assert default
	assert.Equal(t, "*", recorder.Header().Get("Access-Control-Allow-Origin"))

	os.Setenv("KEEDROP_CORS_ORIGINS", "https://keedrop.de")
	router = setupRouter(redis)
	req.Header.Set("Origin", "https://fakedomain.com")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 403, recorder.Code)

	req.Header.Set("Origin", "https://keedrop.de")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	assert.Equal(t, 204, recorder.Code)
	assert.Equal(t, "https://keedrop.de", recorder.Header().Get("Access-Control-Allow-Origin"))
}
