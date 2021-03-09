package main

import (
	"github.com/urfave/cli"
	"plane.watch/lib/tracker"
	"plane.watch/lib/tracker/sbs1"
)

func parseSbs(c *cli.Context) error {
	newFrameFunc := func(line string) tracker.Frame {
		return sbs1.NewFrame(line)
	}
	p, err := produceOutput(c, newFrameFunc)
	if nil != err {
		return err
	}

	ih := tracker.NewInputHandler(tracker.WithVerboseOutput())
	ih.AddProducer(p)
	ih.Wait()

	return writeResult(ih.Tracker, p.outFile)
}
