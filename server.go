// main HTTP server component
package main

import (
  "github.com/gin-gonic/gin"
  "github.com/mediocregopher/radix.v2/pool"
  "github.com/op/go-logging"
  "net/http"
  "strings"
//  "encoding/base64"
  "errors"
)

const pubKeyLen = 32
const nonceLen = 24

var logger = logging.MustGetLogger("keedrop")

type secretData struct {
  PubKey string `json:"pubkey"`
  Nonce string `json:"nonce"`
  Secret string `json:"secret"`
}

// stores the secret in Redis
func storeSecret(redis *pool.Pool, data *secretData) error {
  conn, err := redis.Get()
  if err != nil {
    return errors.New("Redis connection pool exhausted")
  }
  defer redis.Put(conn)

  keyName := redisKey(data.PubKey, data.Nonce)
  if _, err := conn.Cmd("SET", keyName, data.Secret, "NX", "EX", 60*60*24).Str(); err != nil {
    return errors.New("Could not store value")
  }
  return nil
}

func redisKey(pubKey string, nonce string) string {
  // TODO: check pubKey and nonce for plausible length and base64 encoding
  return pubKey + nonce
}

// retrieves the secret from Redis, deleting it at the same time
func retrieveSecret(redis *pool.Pool, pubKey string, nonce string) (string, error) {
  conn, err := redis.Get()
  if err != nil {
    return "", errors.New("Could not connect to Redis")
  }
  defer redis.Put(conn)

  key := redisKey(pubKey, nonce)
  logger.Info("Reading from Redis key ", key)
  if key == "" {
    return "", errors.New("Invalid PubKey/Nonce data")
  }

  conn.PipeAppend("MULTI")
  conn.PipeAppend("GET", key)
  conn.PipeAppend("DEL", key)
  conn.PipeAppend("EXEC")

  for i := 0; i < 3; i++ {
    if err := conn.PipeResp().Err; err != nil {
      logger.Error("Redis error:", err)
      return "", errors.New("Redis communication failed")
    }
  }
  if results, err := conn.PipeResp().Array(); err == nil {
    secret, _ := results[0].Str()
    return secret, nil
  } else {
    logger.Error("Error executing batch:", err)
    return "", errors.New("Redis communication failed")
  }
}

// the Gin handlers all want a Redis connection, too
type redisUsingGinHandler func(*pool.Pool, *gin.Context)

// Gin handler to store a secret
func storePass(redis *pool.Pool, ctx *gin.Context) {
  var secret secretData
  if ctx.BindJSON(&secret) == nil {
    if err := storeSecret(redis, &secret); err == nil {
      ctx.Data(http.StatusNoContent, gin.MIMEJSON, nil)
    } else {
      if (err.Error() == "Could not store value") {
        ctx.JSON(http.StatusConflict, gin.H{"error": "You hit the Jackpot and caused a key collision. Try again."})
      } else {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store secret"})
      }
    }
  } else {
    ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad JSON data"})
  }
}

func splitKeyNonce(str string) (string, string) {
  arr := strings.Split(str, "_")
  if len(arr) != 2 {
    logger.Error("Invalid keynonce: ", str)
    return "", ""
  }
  return arr[0], arr[1]
}

// Gin handler to retrieve (and delete) a secret
func retrievePass(redis *pool.Pool, ctx *gin.Context) {
  pubKey, nonce := splitKeyNonce(ctx.Param("keynonce"))
  if pubKey == "" {
    ctx.JSON(http.StatusBadRequest, gin.H{"error": "Don't know how to interpret data"})
    return
  }
  logger.Info("Reading data for pubkey and nonce: ", pubKey, nonce)
  if secret, err := retrieveSecret(redis, pubKey, nonce); err != nil {
    if err.Error() == "No such secret" {
      ctx.JSON(http.StatusNotFound, gin.H{"error": "No such secret"})
    } else {
      logger.Error("Error reading secret", err)
      ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read secret"})
    }
  } else {
    ctx.JSON(http.StatusOK, gin.H{"secret": secret})
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

  router.POST("/api/store", wrapHandler(redis, storePass))
  router.GET("/api/retrieve/:keynonce", wrapHandler(redis, retrievePass))
  router.StaticFile("/", "./store.html")
  router.StaticFile("/r", "./retrieve.html")
  router.StaticFile("/nacl.js", "./nacl.min.js")
  router.StaticFile("/nacl-util.js", "./nacl-util.min.js")
  router.StaticFile("/keedrop.js", "./keedrop.js")
  router.StaticFile("/styles.css", "./styles.css")
  router.Run()
}
