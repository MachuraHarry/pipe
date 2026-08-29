package object

import (
	"io"
	"os"
	"sync"
)

var (
	fileRegistry   = map[int]*os.File{}
	fileRegistryMu sync.Mutex
	nextFileHandle = 1
)

func init() {
	Builtins = append(Builtins,
		BuiltinInfo{Name: "file_open", Fn: bFileOpen},
		BuiltinInfo{Name: "file_close", Fn: bFileClose},
		BuiltinInfo{Name: "file_read", Fn: bFileRead},
		BuiltinInfo{Name: "file_write", Fn: bFileWrite},
		BuiltinInfo{Name: "file_truncate", Fn: bFileTruncate},
		BuiltinInfo{Name: "file_sync", Fn: bFileSync},
	)
}

func getFile(handle int) (*os.File, bool) {
	fileRegistryMu.Lock()
	defer fileRegistryMu.Unlock()
	f, ok := fileRegistry[handle]
	return f, ok
}

// ---- Random-access file I/O ----

func bFileOpen(args ...Object) Object {
	if len(args) != 2 {
		return err("file_open expects 2 arguments (path, mode)")
	}
	path, ok := args[0].(*String)
	if !ok {
		return err("file_open: path must be a string")
	}
	mode, ok2 := args[1].(*String)
	if !ok2 {
		return err("file_open: mode must be a string (r, w, a, rw, rw+)")
	}

	var flags int
	switch mode.Value {
	case "r":
		flags = os.O_RDONLY
	case "w":
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case "a":
		flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case "rw":
		flags = os.O_RDWR | os.O_CREATE
	case "rw+":
		flags = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	default:
		return err("file_open: unknown mode '" + mode.Value + "' (use r, w, a, rw, rw+)")
	}

	openPath := path.Value
	if mode.Value == "r" {
		if ActiveProfile.Load().Name != "none" {
			var cerr error
			openPath, cerr = ActiveProfile.Load().canonicalRead(path.Value)
			if cerr != nil {
				return err(cerr.Error())
			}
		}
	} else {
		resolved, cerr := checkFSWriteAccess("file_open", path.Value)
		if cerr != nil {
			return cerr
		}
		openPath = resolved
	}

	f, e := os.OpenFile(openPath, flags, 0644)
	if e != nil {
		return err("file_open: " + e.Error())
	}

	fileRegistryMu.Lock()
	handle := nextFileHandle
	nextFileHandle++
	fileRegistry[handle] = f
	fileRegistryMu.Unlock()

	return &Integer{Value: int64(handle)}
}

func bFileClose(args ...Object) Object {
	if len(args) != 1 {
		return err("file_close expects 1 argument (handle)")
	}
	h, ok := ToInt(args[0])
	if !ok {
		return err("file_close: handle must be a number")
	}

	fileRegistryMu.Lock()
	f, exists := fileRegistry[int(h)]
	if exists {
		delete(fileRegistry, int(h))
	}
	fileRegistryMu.Unlock()

	if !exists {
		return err("file_close: invalid handle")
	}
	if e := f.Close(); e != nil {
		return err("file_close: " + e.Error())
	}
	return NILOBJ
}

func bFileRead(args ...Object) Object {
	if len(args) != 3 {
		return err("file_read expects 3 arguments (handle, offset, n)")
	}
	h, ok := ToInt(args[0])
	if !ok {
		return err("file_read: handle must be a number")
	}
	off, ok := ToInt(args[1])
	if !ok {
		return err("file_read: offset must be a number")
	}
	n, ok := ToInt(args[2])
	if !ok {
		return err("file_read: n must be a number")
	}
	if off < 0 {
		return err("file_read: offset must be >= 0")
	}
	if n < 0 {
		return err("file_read: n must be >= 0")
	}
	f, exists := getFile(int(h))
	if !exists {
		return err("file_read: invalid handle")
	}
	buf := make([]byte, n)
	read, e := f.ReadAt(buf, off)
	if e != nil && e != io.EOF {
		return err("file_read: " + e.Error())
	}
	if read < int(n) {
		buf = buf[:read]
	}
	return &Bytes{Value: buf}
}

func bFileWrite(args ...Object) Object {
	if len(args) != 3 {
		return err("file_write expects 3 arguments (handle, offset, data)")
	}
	h, ok := ToInt(args[0])
	if !ok {
		return err("file_write: handle must be a number")
	}
	off, ok := ToInt(args[1])
	if !ok {
		return err("file_write: offset must be a number")
	}
	if off < 0 {
		return err("file_write: offset must be >= 0")
	}
	var data []byte
	switch d := args[2].(type) {
	case *Bytes:
		data = d.Value
	case *String:
		data = []byte(d.Value)
	default:
		return err("file_write: data must be bytes or string")
	}
	f, exists := getFile(int(h))
	if !exists {
		return err("file_write: invalid handle")
	}
	n, e := f.WriteAt(data, off)
	if e != nil {
		return err("file_write: " + e.Error())
	}
	return &Integer{Value: int64(n)}
}

func bFileTruncate(args ...Object) Object {
	if len(args) != 2 {
		return err("file_truncate expects 2 arguments (handle, size)")
	}
	h, ok := ToInt(args[0])
	if !ok {
		return err("file_truncate: handle must be a number")
	}
	size, ok := ToInt(args[1])
	if !ok {
		return err("file_truncate: size must be a number")
	}
	if size < 0 {
		return err("file_truncate: size must be >= 0")
	}
	f, exists := getFile(int(h))
	if !exists {
		return err("file_truncate: invalid handle")
	}
	if e := f.Truncate(size); e != nil {
		return err("file_truncate: " + e.Error())
	}
	return NILOBJ
}

func bFileSync(args ...Object) Object {
	if len(args) != 1 {
		return err("file_sync expects 1 argument (handle)")
	}
	h, ok := ToInt(args[0])
	if !ok {
		return err("file_sync: handle must be a number")
	}
	f, exists := getFile(int(h))
	if !exists {
		return err("file_sync: invalid handle")
	}
	if e := f.Sync(); e != nil {
		return err("file_sync: " + e.Error())
	}
	return NILOBJ
}
