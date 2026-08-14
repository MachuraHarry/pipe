package object

// ---- Channels ----

func bChan(args ...Object) Object {
	if len(args) > 1 {
		return err("chan expects 0-1 arguments (capacity)")
	}
	if len(args) == 0 {
		return NewChannel(0)
	}
	n, ok := ToInt(args[0])
	if !ok {
		return err("chan: capacity must be a number")
	}
	return NewChannel(int(n))
}

func bSend(args ...Object) Object {
	if len(args) != 2 {
		return err("send expects 2 arguments (channel, value)")
	}
	c, ok := args[0].(*Channel)
	if !ok {
		return err("send: first argument must be a channel")
	}
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return err("send: channel is closed")
	}
	c.ch <- args[1]
	return NILOBJ
}

func bRecv(args ...Object) Object {
	if len(args) != 1 {
		return err("recv expects 1 argument (channel)")
	}
	c, ok := args[0].(*Channel)
	if !ok {
		return err("recv: first argument must be a channel")
	}
	v, open := <-c.ch
	if !open {
		return NILOBJ
	}
	return v
}

func bTryRecv(args ...Object) Object {
	if len(args) != 1 {
		return err("try_recv expects 1 argument (channel)")
	}
	c, ok := args[0].(*Channel)
	if !ok {
		return err("try_recv: first argument must be a channel")
	}
	select {
	case v, open := <-c.ch:
		if !open {
			return NILOBJ
		}
		return v
	default:
		return NILOBJ
	}
}

func bTrySend(args ...Object) Object {
	if len(args) != 2 {
		return err("try_send expects 2 arguments (channel, value)")
	}
	c, ok := args[0].(*Channel)
	if !ok {
		return err("try_send: first argument must be a channel")
	}
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return FALSE
	}
	select {
	case c.ch <- args[1]:
		return TRUE
	default:
		return FALSE
	}
}

func bClose(args ...Object) Object {
	if len(args) != 1 {
		return err("close expects 1 argument (channel)")
	}
	c, ok := args[0].(*Channel)
	if !ok {
		return err("close: first argument must be a channel")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return NILOBJ
	}
	c.closed = true
	close(c.ch)
	return NILOBJ
}

func bChanLen(args ...Object) Object {
	if len(args) != 1 {
		return err("chan_len expects 1 argument (channel)")
	}
	c, ok := args[0].(*Channel)
	if !ok {
		return err("chan_len: first argument must be a channel")
	}
	return &Integer{Value: int64(len(c.ch))}
}

func bChanCap(args ...Object) Object {
	if len(args) != 1 {
		return err("chan_cap expects 1 argument (channel)")
	}
	c, ok := args[0].(*Channel)
	if !ok {
		return err("chan_cap: first argument must be a channel")
	}
	return &Integer{Value: int64(cap(c.ch))}
}

// ---- Mutex ----

func bMutex(args ...Object) Object {
	if len(args) != 0 {
		return err("mutex expects 0 arguments")
	}
	return NewMutex()
}

func bLock(args ...Object) Object {
	if len(args) != 1 {
		return err("lock expects 1 argument (mutex)")
	}
	m, ok := args[0].(*Mutex)
	if !ok {
		return err("lock: first argument must be a mutex")
	}
	m.mu.Lock()
	return NILOBJ
}

func bUnlock(args ...Object) Object {
	if len(args) != 1 {
		return err("unlock expects 1 argument (mutex)")
	}
	m, ok := args[0].(*Mutex)
	if !ok {
		return err("unlock: first argument must be a mutex")
	}
	m.mu.Unlock()
	return NILOBJ
}

func bTryLock(args ...Object) Object {
	if len(args) != 1 {
		return err("try_lock expects 1 argument (mutex)")
	}
	m, ok := args[0].(*Mutex)
	if !ok {
		return err("try_lock: first argument must be a mutex")
	}
	return NativeBoolToBoolean(m.mu.TryLock())
}

// ---- Semaphore ----

func bSemaphore(args ...Object) Object {
	if len(args) != 1 {
		return err("semaphore expects 1 argument (count)")
	}
	n, ok := ToInt(args[0])
	if !ok {
		return err("semaphore: count must be a number")
	}
	return NewSemaphore(int(n))
}

func bAcquire(args ...Object) Object {
	if len(args) != 1 {
		return err("acquire expects 1 argument (semaphore)")
	}
	s, ok := args[0].(*Semaphore)
	if !ok {
		return err("acquire: first argument must be a semaphore")
	}
	s.ch <- struct{}{}
	return NILOBJ
}

func bRelease(args ...Object) Object {
	if len(args) != 1 {
		return err("release expects 1 argument (semaphore)")
	}
	s, ok := args[0].(*Semaphore)
	if !ok {
		return err("release: first argument must be a semaphore")
	}
	<-s.ch
	return NILOBJ
}

func bTryAcquire(args ...Object) Object {
	if len(args) != 1 {
		return err("try_acquire expects 1 argument (semaphore)")
	}
	s, ok := args[0].(*Semaphore)
	if !ok {
		return err("try_acquire: first argument must be a semaphore")
	}
	select {
	case s.ch <- struct{}{}:
		return TRUE
	default:
		return FALSE
	}
}
