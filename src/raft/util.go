package raft

import "log"

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

const TestDebug = true
func DTPrintf(format string, a ...interface{}) (n int, err error) {
	if TestDebug {
		log.Printf(format, a...)
	}
	return
}
