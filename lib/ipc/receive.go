package ipc

import (
	"bufio"
	"context"
	"errors"
	"net"
	"os"
	"sync/atomic"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

type AercServer struct {
	listener net.Listener
	handler  Handler
	startup  context.Context
}

func StartServer(handler Handler, startup context.Context) (*AercServer, error) {
	sockpath := xdg.RuntimePath("aerc.sock")
	// remove the socket if it is not connected to a session
	if _, err := ConnectAndExec(nil); err != nil {
		os.Remove(sockpath)
	}
	log.Debugf("Starting Unix server: %s", sockpath)
	l, err := net.Listen("unix", sockpath)
	if err != nil {
		return nil, err
	}
	as := &AercServer{listener: l, handler: handler, startup: startup}
	go as.Serve()

	return as, nil
}

func (as *AercServer) Close() {
	as.listener.Close()
}

var lastId int64 = 0 // access via atomic

func (as *AercServer) Serve() {
	defer log.PanicHandler()

	<-as.startup.Done()

	for {
		conn, err := as.listener.Accept()
		switch {
		case errors.Is(err, net.ErrClosed):
			log.Infof("shutting down UNIX listener")
			return
		case err != nil:
			log.Errorf("ipc: accepting connection failed: %v", err)
			continue
		}

		defer conn.Close()
		clientId := atomic.AddInt64(&lastId, 1)
		log.Debugf("unix:%d accepted connection", clientId)
		scanner := bufio.NewScanner(conn)
		err = conn.SetDeadline(time.Now().Add(1 * time.Minute))
		if err != nil {
			log.Errorf("unix:%d failed to set deadline: %v", clientId, err)
		}
		for scanner.Scan() {
			// allow up to 1 minute between commands
			err = conn.SetDeadline(time.Now().Add(1 * time.Minute))
			if err != nil {
				log.Errorf("unix:%d failed to update deadline: %v", clientId, err)
			}
			msg, err := DecodeRequest(scanner.Bytes())
			log.Tracef("unix:%d got message %s", clientId, scanner.Text())
			if err != nil {
				log.Errorf("unix:%d failed to parse request: %v", clientId, err)
				continue
			}

			response := as.handleMessage(msg)
			result, err := response.Encode()
			if err != nil {
				log.Errorf("unix:%d failed to encode result: %v", clientId, err)
				continue
			}
			_, err = conn.Write(append(result, '\n'))
			if err != nil {
				log.Errorf("unix:%d failed to send response: %v", clientId, err)
				break
			}
		}
		log.Tracef("unix:%d closed connection", clientId)
	}
}

func (as *AercServer) handleMessage(req *Request) *Response {
	err := as.handler.Command(req.Arguments)
	if err != nil {
		return &Response{Error: err.Error()}
	}
	return &Response{}
}
