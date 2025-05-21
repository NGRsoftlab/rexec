package rexec

import (
	"github.com/ngrsoftlab/rexec/config"
	
)

type Client struct {
	*ssh.Client
	Config *config.Config
}

// NewConnreturns new client and error if any
// func NewConn(cfg *config.Config) (*Client, error) {
// 	conn := &Client{
// 		Config: cfg,
// 	}
//
// 	var connectErr error
// 	conn.Client, connectErr = Dial(proto, cfg)
// 	if connectErr != nil {
// 		return nil, err
// 	}
// }
//
// func Dial(proto string, cfg *config.Config) (*ssh.Client, error){
// 	return ssh.Dial(proto, net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port)), &ssh.ClientConfig{
// 		User: cfg.User,
// 		Auth: cfg.Auth
// 	})
// }
