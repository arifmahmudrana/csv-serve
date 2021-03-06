package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/arifmahmudrana/csv-serve/cassandra"
	"github.com/arifmahmudrana/csv-serve/memcached"
	"github.com/arifmahmudrana/csv-serve/redis"
)

const version = "1.0.0"

type application struct {
	infoLog, errorLog *log.Logger
	version           string
	db                cassandra.CassandraRepository
	m                 memcached.MemcachedRepository
	r                 redis.RedisRepository
}

func (app *application) ConnectCassandra() {
	cassandraMaxRetryConnect, err := strconv.Atoi(os.Getenv("CASSANDRA_MAX_RETRY_CONNECT"))
	if err != nil {
		app.errorLog.Fatal(err)
	}

	db, err := cassandra.NewCassandraRepository(
		os.Getenv("CASSANDRA_USER"),
		os.Getenv("CASSANDRA_PASSWORD"),
		cassandraMaxRetryConnect,
		strings.Split(os.Getenv("CASSANDRA_DB_HOST"), ",")...)
	if err != nil {
		app.errorLog.Fatal(err)
	}
	app.db = db
}

func (app *application) ConnectMemcached() {
	m, err := memcached.NewMemcachedRepository(
		strings.Split(os.Getenv("MEMCACHED_SERVER"), ",")...)
	if err != nil {
		app.errorLog.Fatal(err)
	}

	app.m = m
}

func (app *application) ConnectRedis() {
	r, err := redis.NewRedisRepository(
		os.Getenv("REDIS_SERVER"), os.Getenv("REDIS_PREFIX"))
	if err != nil {
		app.errorLog.Fatal(err)
	}

	app.r = r
}

func main() {
	app := &application{
		infoLog:  log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		errorLog: log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
		version:  version,
	}

	app.ConnectCassandra()
	defer app.db.Close()
	app.ConnectMemcached()
	app.ConnectRedis()
	defer app.r.Close()

	// The HTTP Server
	server := &http.Server{
		Addr:              ":8080",
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		// For not calling cancel(https://github.com/grpc/grpc-go/issues/1099)
		shutdownCtx, _ := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				app.errorLog.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			app.errorLog.Fatal(err)
		}
		serverStopCtx()
	}()

	// Run the server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		app.errorLog.Fatal(err)
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}
