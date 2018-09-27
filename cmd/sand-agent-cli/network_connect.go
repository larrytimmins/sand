package main

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func (a *App) NetworkConnect(c *cli.Context) error {
	log := logger.Default()

	client, err := a.sandClient(c)
	if err != nil {
		return err
	}

	if c.String("network") == "" {
		return errors.New("--network flag is mandatory")
	}

	conn, err := client.NetworkConnect(context.Background(), c.String("network"), params.NetworkConnect{
		IP:   c.String("ip"),
		Port: c.String("port"),
	})
	if err != nil {
		return errors.Wrapf(err, "fail to connect to network %v", c.String("network"))
	}

	listener, err := net.Listen("tcp", ":")
	if err != nil {
		return errors.Wrapf(err, "fail to bind a socket")
	}
	defer listener.Close()

	log.Infof("Waiting connection on %v", listener.Addr())

	localConn, err := listener.Accept()
	if err != nil {
		return errors.Wrapf(err, "fail to accept connection")
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer conn.Close()
		_, err := io.Copy(localConn, conn)
		if err != io.EOF && err != nil {
			log.WithError(err).Error("fail to copy data from remote network to local socket")
		}
		log.Info("remote network connection closed")
	}()

	go func() {
		defer wg.Done()
		defer localConn.Close()
		_, err := io.Copy(conn, localConn)
		if err != io.EOF && err != nil {
			log.WithError(err).Error("fail to copy data from remote network to local socket")
		}
		log.Infof("local connection on %v closed", listener.Addr())
	}()

	wg.Wait()
	return nil
}