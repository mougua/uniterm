package session

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTransferConcurrentResume(t *testing.T) {
	task := &TransferTask{ID: "download-1"}
	task.start()
	defer task.done()
	s := NewSFTPSession("transfer-test")
	s.transfers[task.ID] = task
	var events atomic.Int32
	s.SetOnDataCallback(func([]byte) { events.Add(1) })
	if err := s.PauseTransfer(task.ID); err != nil {
		t.Fatal(err)
	}

	finished := make(chan struct{})
	go func() {
		task.waitIfPaused()
		close(finished)
	}()
	var resumes sync.WaitGroup
	for range 20 {
		resumes.Go(func() {
			if err := s.ResumeTransfer(task.ID); err != nil {
				t.Error(err)
			}
		})
	}
	resumes.Wait()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("transfer stayed paused after resume")
	}
	if got := events.Load(); got != 2 {
		t.Fatalf("got %d transfer events, want one pause and one resume", got)
	}
}

func TestSSHDisconnectDuringHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	addr := listener.Addr().(*net.TCPAddr)
	s := NewSSHSession("ssh-test")
	defer s.Disconnect()
	result := make(chan error, 1)
	go func() {
		result <- s.Connect(ConnectionConfig{
			Host: addr.IP.String(), Port: addr.Port, User: "test", Password: "test",
		})
	}()
	if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Connect returned %v, want context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("SSH handshake did not stop after disconnect on port " + strconv.Itoa(addr.Port))
	}
	if got := s.Status(); got != StatusDisconnected {
		t.Fatalf("status = %s, want disconnected", got)
	}
}
