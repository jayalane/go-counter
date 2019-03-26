// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"runtime"
	"strings"
)

func getCallerFunctionName() string {
	// Skip GetCallerFunctionName and the function to get the caller of
	c := getFrame(3).Function
	if strings.Contains(c, "/") {
		cs := strings.Split(c, "/")
		c = cs[len(cs)-1]
	}
	if strings.Contains(c, ".") {
		cs := strings.Split(c, ".")
		c = cs[len(cs)-1]
	}
	return c
}

func getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}
