package web

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/docker/docker/pkg/reexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (c NetworksController) Connect(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx).WithField("network_id", params["id"])

	ip := r.URL.Query().Get("ip")
	port := r.URL.Query().Get("port")
	log = log.WithFields(logrus.Fields{
		"ip":   ip,
		"port": port,
	})
	ctx = logger.ToCtx(ctx, log)
	if ip == "" || port == "" {
		w.WriteHeader(400)
		return errors.New("IP and port are mandatory")
	}

	network, ok, err := c.NetworkRepository.Exists(ctx, params["id"])
	if err != nil {
		return errors.Wrapf(err, "fail to query store")
	} else if !ok {
		w.WriteHeader(404)
		return errors.New("network not found")
	}

	log.Info("hijacking http connection")
	h := w.(http.Hijacker)
	socket, _, err := h.Hijack()
	if err != nil {
		return errors.Wrapf(err, "fail to hijack http connection")
	}
	defer socket.Close()

	fmt.Fprintf(socket, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")

	addr, err := getTempFilename()
	if err != nil {
		return errors.Wrapf(err, "fail to get temp file name for unix socket")
	}

	// Explanation of the following tricky part
	// At first we wanted to forward the socket from the HTTP connection directly
	// to the child process located in the namespace of the sand network, thus
	// able to reach the final target.
	//
	// This method doesn't work as when sand is running with TLS, the connection
	// is a *tls.Conn and there is no way to transfer such socket in a child
	// process.

	// So we need to pass through an intermediary IPC system, here a UNIX socket.
	// Basically the parent process is creating a unix socket server, the child
	// process in the NS of our target is creating a connection to this socket,
	// this connection is linking the remote client connection and the internal
	// sand connection.
	//
	// HTTP client <----- inter-process unix conn -----> connection to target in sand network
	unixSocket, err := net.ListenUnix("unix", &net.UnixAddr{Net: "unix", Name: addr})
	if err != nil {
		return errors.Wrapf(err, "fail to open unix socket")
	}
	defer unixSocket.Close()

	go func() {
		clientSocket, err := unixSocket.AcceptUnix()
		if err != nil {
			log.WithError(err).Error("fail to accept unix connection")
			return
		}
		log.Info("socket unix connection accepted from child process")
		defer clientSocket.Close()

		wg := &sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			defer clientSocket.CloseRead()
			io.Copy(clientSocket, socket)
		}()

		go func() {
			defer wg.Done()
			defer clientSocket.CloseWrite()
			io.Copy(socket, clientSocket)
		}()

		wg.Wait()
	}()

	cmd := &exec.Cmd{
		Path:   reexec.Self(),
		Args:   append([]string{"sc-netns-pipe-socket"}, network.NSHandlePath, ip, port, addr),
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}

	err = cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "fail to pipe socket to %s %s:%s", network.NSHandlePath, ip, port)
	}

	return nil
}

func getTempFilename() (string, error) {
	f, err := ioutil.TempFile("", "sand-connect-")
	if err != nil {
		return "", errors.Wrapf(err, "fail to create temp file")
	}
	addr := f.Name()

	err = f.Close()
	if err != nil {
		return "", errors.Wrapf(err, "fail to close temp file")
	}
	err = os.Remove(addr)
	if err != nil {
		return "", errors.Wrapf(err, "fail to remove temp file")
	}
	return addr, nil
}
