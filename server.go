// main HTTP server component
package main

import (
	"encoding/json"
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

// structure to store the secret in Redis
// only the secret key remains with the sender
type secretData struct {
	PubKey string `json:"pubkey" binding:"required"`
	Nonce  string `json:"nonce" binding:"required"`
	Secret string `json:"secret" binding:"required"`
}

// stores the secret in Redis and returns the key where it can be found
func saveInRedis(redis *pool.Pool, data *secretData) (string, bool) {
	conn, err := redis.Get()
	if err != nil {
		logger.Error("Could not connect to Redis")
		return "", false
	}
	defer redis.Put(conn)

	jsonData, jsonErr := json.Marshal(data)
	if jsonErr != nil {
		logger.Error("Could not marshal secret to JSON.", jsonErr)
		return "", false
	}
	for i := 0; i < maxKeyFindTries; i++ {
		keyName := uniuri.NewLen(mnemoLen)
		if _, err := conn.Cmd("SET", keyName, jsonData, "NX", "EX", defaultLifetime).Str(); err == nil {
			return keyName, true
		} else {
			logger.Error("Could not write secret, probably key collision.", err)
		}
	}
	logger.Error("Could not find key after", maxKeyFindTries, "tries")
	return "", false
}

// retrieves the secret from Redis, deleting it at the same time
func loadFromRedis(redis *pool.Pool, key string) (*secretData, bool) {
	conn, err := redis.Get()
	if err != nil {
		logger.Error("Could not connect to Redis.", err)
		return nil, false
	}
	defer redis.Put(conn)

	conn.PipeAppend("MULTI")
	conn.PipeAppend("GET", key)
	conn.PipeAppend("DEL", key)
	conn.PipeAppend("EXEC")

	// the first 3 commands should only contain "OK" and "QUEUED", no real data
	for i := 0; i < 3; i++ {
		if err := conn.PipeResp().Err; err != nil {
			logger.Error("Redis error.", err)
			return nil, false
		}
	}
	if results, err := conn.PipeResp().Array(); err == nil {
		// array contains the results after MULTI in order
		encodedData, _ := results[0].Bytes()
		if len(encodedData) == 0 { // it means the key wasn't found
			return nil, true
		} else {
			secret := new(secretData)
			if err := json.Unmarshal(encodedData, secret); err == nil {
				return secret, true
			} else {
				logger.Error("Could not unmarshal JSON data: ", encodedData)
				return nil, false
			}
		}
	} else {
		logger.Error("Error executing batch.", err)
		return nil, false
	}
}

// the Gin handlers all want a Redis connection, too
type redisUsingGinHandler func(*pool.Pool, *gin.Context)

// POST /api/secret
func storeSecret(redis *pool.Pool, ctx *gin.Context) {
	var secret secretData
	if ctx.BindJSON(&secret) == nil {
		if keyName, ok := saveInRedis(redis, &secret); ok {
			ctx.JSON(http.StatusOK, gin.H{"key": keyName})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store secret"})
		}
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad JSON data"})
	}
}

// GET /api/secret/:mnemo
func retrieveSecret(redis *pool.Pool, ctx *gin.Context) {
	mnemo := ctx.Param("mnemo")
	logger.Debug("Reading data for mnemo:", mnemo)
	if secret, ok := loadFromRedis(redis, mnemo); !ok {
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
