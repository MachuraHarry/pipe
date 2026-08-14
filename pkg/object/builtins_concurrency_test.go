package object

import (
	"testing"
)

func TestChanSendRecvUnbuffered(t *testing.T) {
	c := NewChannel(0)
	done := make(chan struct{})
	go func() {
		bSend(c, &Integer{Value: 42})
		close(done)
	}()
	got := bRecv(c)
	<-done
	if got.(*Integer).Value != 42 {
		t.Errorf("expected 42, got %s", got.Inspect())
	}
}

func TestChanBuffered(t *testing.T) {
	c := NewChannel(2)
	bSend(c, &Integer{Value: 1})
	bSend(c, &Integer{Value: 2})
	if bChanLen(c).(*Integer).Value != 2 {
		t.Errorf("expected len 2")
	}
	if bChanCap(c).(*Integer).Value != 2 {
		t.Errorf("expected cap 2")
	}
	if bRecv(c).(*Integer).Value != 1 {
		t.Errorf("expected 1")
	}
	if bRecv(c).(*Integer).Value != 2 {
		t.Errorf("expected 2")
	}
}

func TestChanTryRecvEmpty(t *testing.T) {
	c := NewChannel(0)
	if bTryRecv(c) != NILOBJ {
		t.Errorf("expected nil on empty channel")
	}
}

func TestChanTrySendFull(t *testing.T) {
	c := NewChannel(1)
	bSend(c, &Integer{Value: 1})
	if bTrySend(c, &Integer{Value: 2}) != FALSE {
		t.Errorf("expected false on full channel")
	}
	bRecv(c)
	if bTrySend(c, &Integer{Value: 2}) != TRUE {
		t.Errorf("expected true after draining")
	}
}

func TestChanClose(t *testing.T) {
	c := NewChannel(1)
	bSend(c, &Integer{Value: 7})
	bClose(c)
	if bRecv(c).(*Integer).Value != 7 {
		t.Errorf("expected 7 after close (buffered value)")
	}
	if bRecv(c) != NILOBJ {
		t.Errorf("expected nil on closed channel")
	}
	if e := bSend(c, &Integer{Value: 1}); e.Type() != ERROR {
		t.Errorf("expected error on send to closed channel, got %s", e.Inspect())
	}
}

func TestChanCloseIdempotent(t *testing.T) {
	c := NewChannel(0)
	bClose(c)
	bClose(c)
}

func TestMutexTryLock(t *testing.T) {
	m := NewMutex()
	if bTryLock(m) != TRUE {
		t.Errorf("expected true for first try_lock")
	}
	if bTryLock(m) != FALSE {
		t.Errorf("expected false for second try_lock")
	}
	bUnlock(m)
	if bTryLock(m) != TRUE {
		t.Errorf("expected true after unlock")
	}
}

func TestSemaphore(t *testing.T) {
	s := NewSemaphore(2)
	if bTryAcquire(s) != TRUE {
		t.Errorf("expected true")
	}
	if bTryAcquire(s) != TRUE {
		t.Errorf("expected true")
	}
	if bTryAcquire(s) != FALSE {
		t.Errorf("expected false when exhausted")
	}
	bRelease(s)
	if bTryAcquire(s) != TRUE {
		t.Errorf("expected true after release")
	}
}

func TestTypeChecks(t *testing.T) {
	if got := bTypeOf(NewChannel(0)); got.(*String).Value != "CHANNEL" {
		t.Errorf("expected CHANNEL, got %s", got.Inspect())
	}
	if got := bTypeOf(NewMutex()); got.(*String).Value != "MUTEX" {
		t.Errorf("expected MUTEX, got %s", got.Inspect())
	}
	if got := bTypeOf(NewSemaphore(3)); got.(*String).Value != "SEMAPHORE" {
		t.Errorf("expected SEMAPHORE, got %s", got.Inspect())
	}
}

func TestChanSemaphoreArgValidation(t *testing.T) {
	if e := bSend(&Integer{Value: 1}, &Integer{Value: 2}); e.Type() != ERROR {
		t.Errorf("send with non-channel should error")
	}
	if e := bLock(&Integer{Value: 1}); e.Type() != ERROR {
		t.Errorf("lock with non-mutex should error")
	}
	if e := bAcquire(&Integer{Value: 1}); e.Type() != ERROR {
		t.Errorf("acquire with non-semaphore should error")
	}
	if e := bRecv(&Integer{Value: 1}); e.Type() != ERROR {
		t.Errorf("recv with non-channel should error")
	}
	if e := bChan(&String{Value: "x"}); e.Type() != ERROR {
		t.Errorf("chan with non-number capacity should error")
	}
}
