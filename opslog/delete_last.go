package main

import "log"

func getLastOpslog(user string, recursionDelta ...int) string {
	b := 100
	log.Print(b)
	if len(recursionDelta) > 0 {
		b = recursionDelta[0] * 2
	}
	return user
}
