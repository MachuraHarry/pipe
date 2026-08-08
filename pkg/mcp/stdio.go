package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func (s *Server) ServeStdio() error {
	fmt.Fprintln(os.Stderr, s.name+" v"+s.version+" — MCP stdio server running")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		fmt.Fprintf(os.Stderr, "mcp: received %d bytes\n", len(line))
		if len(line) == 0 {
			continue
		}
		reply := s.dispatch(line)
		if reply != nil {
			data, err := json.Marshal(reply)
			if err != nil {
				fmt.Fprintf(os.Stderr, "mcp: marshal error: %v\n", err)
				continue
			}
			fmt.Println(string(data))
		}
	}
	fmt.Fprintln(os.Stderr, "mcp: stdin closed, scanner err: ", scanner.Err())
	return scanner.Err()
}
