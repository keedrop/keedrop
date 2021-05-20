// main HTTP server component
package main

import (
	"encoding/json"
	"github.com/dchest/uniuri"
	"github.com/fvbock/endless"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/mediocregopher/radix/v3"
	"github.com/op/go-logging"
	"net/http"
	"os"
	"strings"
)

const (
	mnemoLen                = 10
	defaultLifetime         = 60 * 60 * 24
	maxMnemoFindTries       = 10
	secretsStoredCounter    = "KeeDropStoredKeysCounter"
	secretsRetrievedCounter = "KeeDropRetrievedKeysCounter"
)

var logger = logging.MustGetLogger("keedrop")

func corsConfig() cors.Config {
	return cors.Config{
		AllowOrigins: getCorsOrigins(),
		AllowMethods: []string{"POST", "GET"},
		AllowHeaders: []string{"Content-Type"},
	}
}

func mapSlice(src []string, f func(string) string) []string {
	mapped := make([]string, len(src))
	for i, v := range src {
		mapped[i] = f(v)
	}
	return mapped
}

func getCorsOrigins() []string {
	origins := os.Getenv("KEEDROP_CORS_ORIGINS")
	if len(origins) == 0 {
		return []string{"*"}
	}
	sliced := strings.Split(origins, ",")
	return mapSlice(sliced, strings.TrimSpace)
}

// structure to store the secret in Redis
// only the secret key remains with the sender
// secret for test.json: Lz5DP4grKMN9efoL9dt!S81X7AFGhin3OHDgbB8qcqQ=
type secretData struct {
	PubKey string `json:"pubkey" binding:"required"`
	Nonce  string `json:"nonce" binding:"required"`
	Secret string `json:"secret" binding:"required"`
}

func increaseCounter(redis *radix.Pool, counterName string) {
	if err := (*redis).Do(radix.Cmd(nil, "INCR", counterName)); err != nil {
		logger.Error("Could not increase counter", err)
	}
}

// stores the secret in Redis and returns the key(mnemo) where it can be found
func saveInRedis(redis *radix.Pool, data *secretData) (string, bool) {
	jsonData, jsonErr := json.Marshal(data)
	if jsonErr != nil {
		logger.Error("Could not marshal secret to JSON.", jsonErr)
		return "", false
	}
	for i := 0; i < maxMnemoFindTries; i++ {
		mnemo := uniuri.NewLen(mnemoLen)
		if err := (*redis).Do(radix.FlatCmd(nil, "SET", mnemo, jsonData, "NX", "EX", defaultLifetime)); err == nil {
			increaseCounter(redis, secretsStoredCounter)
			return mnemo, true
		} else {
			logger.Error("Could not write secret, probably key collision.", err)
		}
	}
	logger.Error("Could not find unused mnemo after", maxMnemoFindTries, "tries")
	return "", false
}

// retrieves the secret from Redis, deleting it at the same time
func loadFromRedis(redis *radix.Pool, mnemo string) (*secretData, bool) {

	var encodedData string

	if err := (*redis).Do(radix.WithConn(mnemo, func(conn radix.Conn) error {
		if err := conn.Do(radix.Cmd(nil, "MULTI")); err != nil {
			return err
		}

		var err error
		defer func() {
			if err != nil {
				conn.Do(radix.Cmd(nil, "DISCARD"))
			}
		}()

		if err = conn.Do(radix.Cmd(nil, "GET", mnemo)); err != nil {
			return err
		}
		if err = conn.Do(radix.Cmd(nil, "DEL", mnemo)); err != nil {
			return err
		}
		var result []string
		if err = conn.Do(radix.Cmd(&result, "EXEC")); err != nil {
			return err
		}
		encodedData = result[0]
		return nil
	})); err != nil {
		logger.Error("Failed to execute command batch")
		return nil, false
	}

	if len(encodedData) == 0 { // it means the secret wasn't found
		return nil, true
	} else {
		secret := new(secretData)
		if err := json.Unmarshal([]byte(encodedData), secret); err == nil {
			increaseCounter(redis, secretsRetrievedCounter)
			return secret, true
		} else {
			logger.Error("Could not unmarshal JSON data: ", encodedData)
			return nil, false
		}
	}
}

// the Gin handlers all want a Redis connection, too
type redisUsingGinHandler func(*radix.Pool, *gin.Context)

// POST /api/secret
func storeSecret(redis *radix.Pool, ctx *gin.Context) {
	var secret secretData
	if ctx.BindJSON(&secret) == nil {
		if mnemo, ok := saveInRedis(redis, &secret); ok {
			ctx.JSON(http.StatusOK, gin.H{"mnemo": mnemo})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store secret"})
		}
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad JSON data"})
	}
}

// GET /api/secret/:mnemo
func retrieveSecret(redis *radix.Pool, ctx *gin.Context) {
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
func wrapHandler(redis *radix.Pool, wrapped redisUsingGinHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wrapped(redis, ctx)
	}
}

func redisConnectionString() string {
	if connectionString := os.Getenv("KEEDROP_REDIS"); len(connectionString) > 0 {
		return connectionString
	} else {
		return "redis://localhost:6379/0"
	}
}

func listenPort() string {
	if listenPort := os.Getenv("KEEDROP_PORT"); len(listenPort) > 0 {
		return listenPort
	} else {
		return ":8080"
	}
}

func setupRouter(redis *radix.Pool) *gin.Engine {
	router := gin.Default()

	router.Use(static.Serve("/", static.LocalFile("./_site", true)))

	router.Use(cors.New(corsConfig()))

	router.POST("/api/secret", wrapHandler(redis, storeSecret))
	router.GET("/api/secret/:mnemo", wrapHandler(redis, retrieveSecret))

	return router
}

// application entry point
func main() {
	redisUri := redisConnectionString()
	redis, err := radix.NewPool("tcp", redisUri, 10)
	if err != nil {
		logger.Fatal("Cannot connect to Redis on", redisUri)
	}
	router := setupRouter(redis)
	endless.ListenAndServe(listenPort(), router)
}
