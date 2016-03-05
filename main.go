// main HTTP server component
package main

import (
  "github.com/gin-gonic/gin"
  "github.com/mediocregopher/radix.v2/pool"
  "github.com/op/go-logging"
  "net/http"
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

  keyName := data.PubKey + data.Nonce
  logger.Debug("Writing key", keyName)
  if _, err := conn.Cmd("SET", keyName, data.Secret, "NX", "EX", 60*60*24).Str(); err != nil {
    return errors.New("Could not store value")
  }
  return nil
}

// retrieves the secret from Redis, deleting it at the same time
func retrieveSecret(redis *pool.Pool, pubKeyNonce string) (string, error) {
  if len(pubKeyNonce) != (pubKeyLen + nonceLen) {
    return "", errors.New("Invalid key length")
  }

  conn, err := redis.Get()
  if err != nil {
    return "", errors.New("Redis connection pool exhausted")
  }
  defer redis.Put(conn)

  logger.Debug("Reading key", pubKeyNonce)
  if secret, err := conn.Cmd("GET", pubKeyNonce).Str(); err != nil {
    logger.Error("Could not read secret: ", err)
    return "", errors.New("No such secret")
  } else {
    if _, delErr := conn.Cmd("DEL", pubKeyNonce).Str(); delErr != nil {
      // https://github.com/mediocregopher/radix.v2/issues/23
      if delErr.Error() != "wrong type" {
        logger.Error("Could not delete secret: ", err)
      }
    }
    return secret, nil
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

// Gin handler to retrieve (and delete) a secret
func retrievePass(redis *pool.Pool, ctx *gin.Context) {
  keyNonce := ctx.Param("keynonce")
  if secret, err := retrieveSecret(redis, keyNonce); err != nil {
    if err.Error() == "No such secret" {
      ctx.JSON(http.StatusNotFound, gin.H{"error": "No such secret"})
    } else {
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
  router.Run()
}
