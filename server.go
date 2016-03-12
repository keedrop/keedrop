// main HTTP server component
package main

import (
	"encoding/json"
	"errors"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/op/go-logging"
	"net/http"
)

const (
	mnemoLen        = 10
	defaultLifetime = 60 * 60 * 24
	maxKeyFindTries = 10
)

var logger = logging.MustGetLogger("keedrop")

type secretData struct {
	PubKey string `json:"pubkey" binding:"required"`
	Nonce  string `json:"nonce" binding:"required"`
	Secret string `json:"secret" binding:"required"`
}

// stores the secret in Redis
func saveInRedis(redis *pool.Pool, data *secretData) (string, error) {
	conn, err := redis.Get()
	if err != nil {
		return "", errors.New("Could not connect to Redis")
	}
	defer redis.Put(conn)

	jsonData, jsonErr := json.Marshal(data)
	if jsonErr != nil {
		return "", errors.New("Could not encode to JSON")
	}
	for i := 0; i < maxKeyFindTries; i++ {
		keyName := uniuri.NewLen(mnemoLen)
		if _, err := conn.Cmd("SET", keyName, jsonData, "NX", "EX", defaultLifetime).Str(); err == nil {
			return keyName, nil
		}
	}
	return "", errors.New("Could not generate key")
}

// retrieves the secret from Redis, deleting it at the same time
func loadFromRedis(redis *pool.Pool, key string) (*secretData, error) {
	conn, err := redis.Get()
	if err != nil {
		return nil, errors.New("Could not connect to Redis")
	}
	defer redis.Put(conn)

	logger.Info("Reading from Redis key ", key)

	conn.PipeAppend("MULTI")
	conn.PipeAppend("GET", key)
	conn.PipeAppend("DEL", key)
	conn.PipeAppend("EXEC")

	for i := 0; i < 3; i++ {
		if err := conn.PipeResp().Err; err != nil {
			logger.Error("Redis error:", err)
			return nil, errors.New("Redis communication failed")
		}
	}
	if results, err := conn.PipeResp().Array(); err == nil {
		encodedData, _ := results[0].Bytes()
		if len(encodedData) == 0 {
			return nil, nil
		} else {
			secret := new(secretData)
			json.Unmarshal(encodedData, secret)
			return secret, nil
		}
	} else {
		logger.Error("Error executing batch:", err)
		return nil, errors.New("Redis communication failed")
	}
}

// the Gin handlers all want a Redis connection, too
type redisUsingGinHandler func(*pool.Pool, *gin.Context)

// Gin handler to store a secret
func storeSecret(redis *pool.Pool, ctx *gin.Context) {
	var secret secretData
	if ctx.BindJSON(&secret) == nil {
		if keyName, err := saveInRedis(redis, &secret); err == nil {
			ctx.JSON(http.StatusOK, gin.H{"key": keyName})
		} else {
			if err.Error() == "Could not store value" {
				ctx.JSON(http.StatusConflict, gin.H{"error": "You hit the Jackpot and caused a key collision. Try again."})
			} else {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store secret"})
			}
		}
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad JSON data"})
	}
}

// Gin handler to retrieve (and delete) a secret
func retrieveSecret(redis *pool.Pool, ctx *gin.Context) {
	mnemo := ctx.Param("mnemo")
	logger.Info("Reading data for mnemo:", mnemo)
	if secret, err := loadFromRedis(redis, mnemo); err != nil {
		logger.Error("Error reading secret", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read secret"})
	} else {
		if secret == nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "No such secret"})
		} else {
			ctx.JSON(http.StatusOK, secret)
		}
	}
}

// ensures that the Gin handler function receives a Redis connection, too
func wrapHandler(redis *pool.Pool, wrapped redisUsingGinHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wrapped(redis, ctx)
	}
}

// application entry point
func main() {
	redis, err := pool.New("tcp", "localhost:6379", 10)
	if err != nil {
		logger.Fatal("Cannot connect to Redis")
	}
	router := gin.Default()

	router.POST("/api/secret", wrapHandler(redis, storeSecret))
	router.GET("/api/secret/:mnemo", wrapHandler(redis, retrieveSecret))
	router.StaticFile("/", "./store.html")
	router.StaticFile("/r", "./retrieve.html")
	router.StaticFile("/nacl.js", "./nacl.min.js")
	router.StaticFile("/nacl-util.js", "./nacl-util.min.js")
	router.StaticFile("/keedrop.js", "./keedrop.js")
	router.StaticFile("/styles.css", "./styles.css")
	router.Run()
}
