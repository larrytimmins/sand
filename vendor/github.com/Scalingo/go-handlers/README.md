Custom Router and Handler
=========================

## Advantages

### Injection of middlewares

```go
type Middleware interface {
  Apply(f HandlerFunc) HandlerFunc
}
type MiddlewareFunc func(HandlerFunc) HandlerFunc
```

Middleware as a struct:

```go
type MyMiddleware struct {
  SomeData string
}

func(m *MyMiddleware) Apply(h HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		// Do something before handler
		h(w, r, vars)
		// Do something after handler
	}
}
```

Or as a function

```go
func MyFunctionMiddleware(h HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		// Do something before handler
		h(w, r, vars)
		// Do something after handler
	}
}
```

Add them to your router with:

```go
router.Use(MyMiddleware)
router.Use(MiddlewareFunc(MyMiddleware))
```

> Middlewares have to be setup before setting the handlers.

### Error propagation in the middlewares

The handler returns an error which can be read/managed by
the middlwares. So you can add your __Airbrake__ or __Rollbar__
notification as a simple middleware.

### Testable handlers based on _Gorilla Muxer_

`mux.Vars(req)` → `vars map[string]string` argument of Handler

## Included middlewares

### Logging Middleware

```go
logger := logrus.New()
middleware := NewLoggingMiddleware(logger)
router.Use(middleware)
```

That being said there when `NewRouter` it creates a LoggingMiddleware by
default.

### Cors Middleware

```go
router.Use(MiddlewareFunc(NewCorsMiddleware))
```

### Error Middleware

Thie middleware writes in the logs with the `Error` log level.
To send logs to rollbar, ensure your logger is properly configured
with the rollbar hook.


```go
import (
  "gopkg.in/Scalingo/logrus-rollbar.v1"
)

logger := logger.Default(logrus_rollbar.New(0))
router := handlers.NewRouter(logger)
router.Use(MiddlewareFunc(ErrorHandler))
```