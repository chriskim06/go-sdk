package web

import (
	"net/http"
	"testing"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/logger"
)

func TestHealthz(t *testing.T) {
	assert := assert.New(t)

	appLog := logger.New().WithFlags(logger.AllFlags())
	defer appLog.Close()

	app := New().WithBindAddr("127.0.0.1:0").WithLogger(appLog).WithConfig(MustNewConfigFromEnv())
	defer app.Shutdown()

	appStarted := make(chan struct{})
	appLog.Listen(AppStartComplete, "default", NewAppEventListener(func(aes *AppEvent) {
		close(appStarted)
	}))

	hzLog := logger.New().WithFlags(logger.AllFlags())
	defer hzLog.Close()

	hz := NewHealthz(app).WithLogger(hzLog).WithGracePeriodSeconds(0)
	defer hz.Shutdown()
	hz.WithDefaultHeader("key", "secure")
	assert.NotEmpty(hz.DefaultHeaders())

	assert.NotNil(hz.Hosted())
	assert.False(app.Latch().IsRunning())

	go hz.Start()
	<-hz.hosted.NotifyStarted()
	<-hz.self.NotifyStarted()

	assert.True(hz.hosted.Latch().IsRunning())
	assert.True(hz.self.Latch().IsRunning())

	assert.NotNil(hz.hosted.Listener())
	assert.NotNil(hz.self.Listener())

	healthzRes, err := http.Get("http://" + hz.self.Listener().Addr().String() + "/healthz")
	assert.Nil(err)
	assert.Equal(http.StatusOK, healthzRes.StatusCode)
	assert.Equal("secure", healthzRes.Header.Get("key"))

	app.Shutdown()
	<-app.NotifyShutdown()

	healthzRes, err = http.Get("http://" + hz.self.Listener().Addr().String() + "/healthz")
	assert.Nil(err)
	assert.Equal(http.StatusInternalServerError, healthzRes.StatusCode)

	notfoundRes, err := http.Get("http://" + hz.self.Listener().Addr().String() + "/adfasdfa")
	assert.Nil(err)
	assert.Equal(http.StatusNotFound, notfoundRes.StatusCode)
}

func TestHealthzProperties(t *testing.T) {
	assert := assert.New(t)

	hz := NewHealthz(nil)
	assert.False(hz.RecoverPanics())
	hz.WithRecoverPanics(true)
	assert.True(hz.RecoverPanics())

	assert.Nil(hz.Logger())
	hz.WithLogger(logger.None())
	assert.NotNil(hz.Logger())
}
