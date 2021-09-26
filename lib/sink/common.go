package sink

import (
	"io"
	"os"
	"sync"
)

type (
	Config struct {
		host, port string
		secure bool

		vhost string
		user, pass string
		queue map[string]string

		out io.WriteCloser
		waiter sync.WaitGroup

		logLocation bool
		sourceTag         string
		messageTtlSeconds int
	}
	Option func(*Config)
)

func WithHost(host, port string) Option {
	return func(conf *Config) {
		conf.host = host
		conf.port = port
	}
}
func WithUserPass(user, pass string) Option {
	return func(conf *Config) {
		conf.user = user
		conf.pass = pass
	}
}

func WithLogOutput(out io.WriteCloser) Option {
	return func(config *Config) {
		config.out = out
	}
}

func WithoutLoggingLocation() Option {
	return func(config *Config) {
		config.logLocation = false
	}
}

func WithSourceTag(tag string) Option {
	return func(config *Config) {
		config.sourceTag = tag
	}
}

func WithMessageTtl(ttl int) Option {
	return func(config *Config) {
		if ttl >= 0 {
			config.messageTtlSeconds = ttl
		}
	}
}

func WithLogFile(file string) Option {
	return func(config *Config) {
		f, err := os.Create(file)
		if nil != err {
			println("Cannot open file: ", file)
			return
		}
		config.out = f
	}
}

func (c *Config) Finish() {
	c.waiter.Wait()
	if nil != c.out {
		_ = c.out.Close()
	}
}