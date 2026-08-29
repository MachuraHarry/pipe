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
	path, cerr := checkFSWriteAccess("append_file", p.Value)
	if cerr != nil {
		return cerr
	}
	f, e := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	path := p.Value
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		path, cerr = ActiveProfile.Load().canonicalRead(p.Value)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	data, e := os.ReadFile(path)
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
	path := p.Value
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		path, cerr = ActiveProfile.Load().canonicalRead(p.Value)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	_, e := os.Stat(path)
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
	path, cerr := checkFSWriteAccess("file_delete", p.Value)
	if cerr != nil {
		return cerr
	}
	if e := os.Remove(path); e != nil {
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
	srcPath := src.Value
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		srcPath, cerr = ActiveProfile.Load().canonicalRead(src.Value)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	dstPath, cerr := checkFSWriteAccess("file_move", dst.Value)
	if cerr != nil {
		return cerr
	}
	if e := os.Rename(srcPath, dstPath); e != nil {
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
	srcPath := src.Value
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		srcPath, cerr = ActiveProfile.Load().canonicalRead(src.Value)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	dstPath, cerr := checkFSWriteAccess("file_copy", dst.Value)
	if cerr != nil {
		return cerr
	}
	srcFile, e := os.Open(srcPath)
	if e != nil {
		return err("file_copy: " + e.Error())
	}
	defer srcFile.Close()
	dstFile, e := os.Create(dstPath)
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
	path := p.Value
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		path, cerr = ActiveProfile.Load().canonicalRead(p.Value)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	info, e := os.Stat(path)
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
	path := p.Value
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		path, cerr = ActiveProfile.Load().canonicalRead(p.Value)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	info, e := os.Stat(path)
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
	dirPath := path
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		dirPath, cerr = ActiveProfile.Load().canonicalRead(path)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	entries, e := os.ReadDir(dirPath)
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
	path, cerr := checkFSWriteAccess("make_dir", p.Value)
	if cerr != nil {
		return cerr
	}
	if e := os.MkdirAll(path, 0755); e != nil {
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
	path, cerr := checkFSWriteAccess("remove_dir", p.Value)
	if cerr != nil {
		return cerr
	}
	if e := os.RemoveAll(path); e != nil {
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
