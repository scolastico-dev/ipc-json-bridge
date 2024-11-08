package main

import (
	"net"
	"syscall"
)

func getPeerPID(conn net.Conn) int {
	file, err := conn.(*net.UnixConn).File()
	if err != nil {
		return 0
	}
	defer file.Close()
	
	ucred, err := syscall.GetsockoptUcred(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return 0
	}
	return int(ucred.Pid)
}
