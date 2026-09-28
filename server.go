// main HTTP server component
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/fvbock/endless"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/mediocregopher/radix/v3"
	"github.com/op/go-logging"
	"net/http"
	"os"
	"time"
	"strings"
)

const (
	mnemoLen                = 10
	maxRequestBodyBytes     = 64 * 1024
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
		MaxAge: 12 * time.Hour,
	}
}

func mapSlice(src []string, f func(string) string) []string {
	mapped := make([]string, len(src))
	for i, v := range src {
		mapped[i] = f(v)
	}
	return mapped
}

// the site calls the API on its own origin, so CORS is off unless origins are configured
func getCorsOrigins() []string {
	origins := os.Getenv("KEEDROP_CORS_ORIGINS")
	if len(origins) == 0 {
		return nil
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
			logger.Error("Could not unmarshal secret JSON data:", err)
			return nil, false
		}
	}
}

// the Gin handlers all want a Redis connection, too
type redisUsingGinHandler func(*radix.Pool, *gin.Context)

// POST /api/secret
func storeSecret(redis *radix.Pool, ctx *gin.Context) {
	var secret secretData
	var tooLarge *http.MaxBytesError
	if err := ctx.ShouldBindJSON(&secret); errors.As(err, &tooLarge) {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Secret too large"})
	} else if err == nil {
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

// only proxies listed here may set X-Forwarded-For, so clients can't spoof their IP
func trustedProxies() []string {
	if proxies := os.Getenv("KEEDROP_TRUSTED_PROXIES"); len(proxies) > 0 {
		return mapSlice(strings.Split(proxies, ","), strings.TrimSpace)
	}
	return []string{"127.0.0.1", "::1"}
}

// the site only loads its own scripts, styles and API, plus the badge images
const contentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; " +
	"img-src 'self' https://img.shields.io; connect-src 'self'; manifest-src 'self'; " +
	"base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// sets browser security headers on every response, pages and API alike
func securityHeaders(ctx *gin.Context) {
	header := ctx.Writer.Header()
	header.Set("Strict-Transport-Security", "max-age=31536000")
	header.Set("Content-Security-Policy", contentSecurityPolicy)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	ctx.Next()
}

// secrets must never end up in a browser or proxy cache
func noStore(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Next()
}

// rejects request bodies larger than maxRequestBodyBytes
func limitRequestBody(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxRequestBodyBytes)
	ctx.Next()
}

// like Gin's default request log, but without client IPs and with the
// route pattern instead of the path, so secret mnemos never end up in logs
func requestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		path := param.Path
		if strings.HasPrefix(path, "/api/secret/") {
			path = "/api/secret/:mnemo"
		}
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %-7s %#v\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.Method,
			path,
		)
	})
}

func setupRouter(redis *radix.Pool) *gin.Engine {
	router := gin.New()
	router.Use(requestLogger(), gin.Recovery(), securityHeaders)
	if err := router.SetTrustedProxies(trustedProxies()); err != nil {
		logger.Fatal("Invalid KEEDROP_TRUSTED_PROXIES:", err)
	}

	router.Use(static.Serve("/", static.LocalFile("./_site", true)))

	if len(getCorsOrigins()) > 0 {
		router.Use(cors.New(corsConfig()))
	}

	api := router.Group("/api", noStore)
	if limit := rateLimit(); limit > 0 {
		api.Use(newRateLimiter(limit).middleware())
	}
	api.POST("/secret", limitRequestBody, wrapHandler(redis, storeSecret))
	api.GET("/secret/:mnemo", wrapHandler(redis, retrieveSecret))

  router.NoRoute(func(c *gin.Context) {
    path := strings.TrimSuffix(c.Request.URL.Path, "/")
    htmlPath := "./_site" + path + ".html"

    if _, err := os.Stat(htmlPath); err == nil {
      c.File(htmlPath)
      return
    }

    c.Status(404)
  })

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
