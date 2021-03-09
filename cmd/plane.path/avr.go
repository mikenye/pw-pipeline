package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"plane.watch/lib/tracker"
	"plane.watch/lib/tracker/mode_s"
	"sync"
	"time"
)

func parseAvr(c *cli.Context) error {
	stdOut := c.GlobalBool("stdout")
	verbose := c.GlobalBool("v")

	var outFileName string
	var dataFiles []string
	if stdOut {
		dataFiles = c.Args()
	} else if c.NArg() > 1 {
		outFileName = c.Args().First()
		dataFiles = c.Args()[1:]
	}

	if 0 == len(dataFiles) {
		return errors.New("please specify datafiles to use")
	}

	if verbose {
		tracker.SetLoggerOutput(os.Stdout)
	}


	inputLines, errChan := readFiles(dataFiles)

	jobChan := make(chan mode_s.ReceivedFrame, 1000)
	resultChan := make(chan *mode_s.Frame, 1000)
	errorChan := make(chan error, 1000)
	exitChan := make(chan bool)

	go handleReceived(resultChan)
	go handleErrors(errChan, verbose)
	go handleErrors(errorChan, verbose)
	if !verbose {
		tracker.SetLoggerOutput(ioutil.Discard)
	}
	go mode_s.DecodeStringWorker(jobChan, resultChan, errorChan)

	go func() {
		var ts time.Time
		for line := range inputLines {
			ts = ts.Add(500 * time.Millisecond)
			jobChan <- mode_s.ReceivedFrame{Time: ts, Frame: line}
		}

		for len(jobChan) > 0 {
			time.Sleep(time.Second)
		}

		exitChan <- true
	}()

	select {
	case <-exitChan:
		close(jobChan)
		close(resultChan)
		close(errorChan)
	}

	fmt.Fprintf(os.Stderr, "We have %d points tracked\n", tracker.PointCounter)

	return writeResult(outFileName)
}

var lastSeenMap sync.Map
func handleReceived(results chan *mode_s.Frame) {
	var resultCounter int
	for {
		select {
		case frame := <-results:
			lastSeen, _ := lastSeenMap.LoadOrStore(frame.ICAOAddr(), time.Now().Add(-time.Hour))
			if frame.VelocityValid() {

			} else {
				lastSeen = time.Time(lastSeen).Add(time.Second)
			}

			resultCounter++
			plane := tracker.HandleModeSFrame(frame)
			if nil != plane {
				// whee plane changed - now has it moved from its last position?
				if resultCounter%1000 == 0 {
					fmt.Fprintf(os.Stderr, "Results: %d (buf %d)\r", resultCounter, len(results))
				}
			}
		}
	}
}
func handleErrors(errors chan error, verbose bool) {
	for {
		select {
		case err := <-errors:
			if verbose && nil != err {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}
