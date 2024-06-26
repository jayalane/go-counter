// -*- tab-width: 2 -*-

package counters

import (
	"runtime"
	"strings"
)

func getCallerFunctionName() string {
	// Skip GetCallerFunctionName and the function to get the caller of
	c := getFrame(3).Function // nolint:mnd
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
	targetFrameIndex := skipFrames + 2 //nolint:mnd

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2) //nolint:mnd
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
