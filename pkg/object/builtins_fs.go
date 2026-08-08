package object

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---- File System ----

func bAppendFile(args ...Object) Object {
	if len(args) != 2 {
		return err("append_file expects 2 arguments (path, content)")
	}
	p, ok := args[0].(*String)
	c, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("append_file: path and content must be strings")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	f, e := os.OpenFile(p.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return err("append_file: " + e.Error())
	}
	defer f.Close()
	if _, e := f.WriteString(c.Value); e != nil {
		return err("append_file: " + e.Error())
	}
	return NILOBJ
}

func bReadLines(args ...Object) Object {
	if len(args) != 1 {
		return err("read_lines expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("read_lines: path must be a string")
	}
	data, e := os.ReadFile(p.Value)
	if e != nil {
		return err("read_lines: " + e.Error())
	}
	lines := strings.Split(string(data), "\n")
	elems := make([]Object, len(lines))
	for i, l := range lines {
		elems[i] = &String{Value: l}
	}
	return &List{Elements: elems}
}

func bFileExists(args ...Object) Object {
	if len(args) != 1 {
		return err("file_exists expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_exists: path must be a string")
	}
	_, e := os.Stat(p.Value)
	return NativeBoolToBoolean(e == nil)
}

func bFileDelete(args ...Object) Object {
	if len(args) != 1 {
		return err("file_delete expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_delete: path must be a string")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowFS {
		return sandboxBlock("file_delete (filesystem write)")
	}
	if e := os.Remove(p.Value); e != nil {
		return err("file_delete: " + e.Error())
	}
	return NILOBJ
}

func bFileMove(args ...Object) Object {
	if len(args) != 2 {
		return err("file_move expects 2 arguments (source, destination)")
	}
	src, ok := args[0].(*String)
	dst, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("file_move: paths must be strings")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(dst.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	if e := os.Rename(src.Value, dst.Value); e != nil {
		return err("file_move: " + e.Error())
	}
	return NILOBJ
}

func bFileCopy(args ...Object) Object {
	if len(args) != 2 {
		return err("file_copy expects 2 arguments (source, destination)")
	}
	src, ok := args[0].(*String)
	dst, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("file_copy: paths must be strings")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanRead(src.Value); canErr != nil {
			return err(canErr.Error())
		}
		if canErr := ActiveProfile.CanWrite(dst.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	srcFile, e := os.Open(src.Value)
	if e != nil {
		return err("file_copy: " + e.Error())
	}
	defer srcFile.Close()
	dstFile, e := os.Create(dst.Value)
	if e != nil {
		return err("file_copy: " + e.Error())
	}
	defer dstFile.Close()
	if _, e := io.Copy(dstFile, srcFile); e != nil {
		return err("file_copy: " + e.Error())
	}
	return NILOBJ
}

func bFileSize(args ...Object) Object {
	if len(args) != 1 {
		return err("file_size expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_size: path must be a string")
	}
	info, e := os.Stat(p.Value)
	if e != nil {
		return err("file_size: " + e.Error())
	}
	return &Integer{Value: info.Size()}
}

func bFileType(args ...Object) Object {
	if len(args) != 1 {
		return err("file_type expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_type: path must be a string")
	}
	info, e := os.Stat(p.Value)
	if e != nil {
		return err("file_type: " + e.Error())
	}
	if info.IsDir() {
		return &String{Value: "dir"}
	}
	return &String{Value: "file"}
}

func bListDir(args ...Object) Object {
	path := "."
	if len(args) >= 1 {
		p, ok := args[0].(*String)
		if !ok {
			return err("list_dir: path must be a string")
		}
		path = p.Value
	}
	entries, e := os.ReadDir(path)
	if e != nil {
		return err("list_dir: " + e.Error())
	}
	elems := make([]Object, len(entries))
	for i, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		elems[i] = &String{Value: name}
	}
	sort.Slice(elems, func(i, j int) bool {
		return elems[i].(*String).Value < elems[j].(*String).Value
	})
	return &List{Elements: elems}
}

func bMakeDir(args ...Object) Object {
	if len(args) != 1 {
		return err("make_dir expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("make_dir: path must be a string")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	if e := os.MkdirAll(p.Value, 0755); e != nil {
		return err("make_dir: " + e.Error())
	}
	return NILOBJ
}

func bRemoveDir(args ...Object) Object {
	if len(args) != 1 {
		return err("remove_dir expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("remove_dir: path must be a string")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowFS {
		return sandboxBlock("remove_dir (filesystem write)")
	}
	if e := os.RemoveAll(p.Value); e != nil {
		return err("remove_dir: " + e.Error())
	}
	return NILOBJ
}

func bPathJoin(args ...Object) Object {
	parts := make([]string, len(args))
	for i, a := range args {
		s, ok := a.(*String)
		if !ok {
			return err("path_join: all arguments must be strings")
		}
		parts[i] = s.Value
	}
	return &String{Value: filepath.Join(parts...)}
}

func bPathBase(args ...Object) Object {
	if len(args) != 1 {
		return err("path_base expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_base expects a string")
	}
	return &String{Value: filepath.Base(s.Value)}
}

func bPathDir(args ...Object) Object {
	if len(args) != 1 {
		return err("path_dir expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_dir expects a string")
	}
	return &String{Value: filepath.Dir(s.Value)}
}

func bPathExt(args ...Object) Object {
	if len(args) != 1 {
		return err("path_ext expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_ext expects a string")
	}
	return &String{Value: filepath.Ext(s.Value)}
}
