package main

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/kpawlik/geojson"
	"github.com/urfave/cli"
	"os"
	"plane.watch/lib/tracker"
	"strings"
	"time"
)

type (
	producer struct {
		outFile   string
		dataFiles []string
		verbose   bool

		input   chan string
		errChan chan error
		logChan chan tracker.LogEvent
		out     chan tracker.Event

		newFrame frameFunc
	}

	frameFunc func(string) *tracker.FrameEvent
)

func main() {
	app := cli.NewApp()

	app.Version = "v0.0.2"
	app.Name = "Plane Watch flight Path Renderer"
	app.Usage = "Reads AVR frames or SBS1 data from a file and generates a GeoJSON file"
	app.Authors = []cli.Author{
		{Name: "Jason Playne", Email: "jason@jasonplayne.com"},
	}
	cli.VersionFlag = cli.BoolFlag{Name: "version, V"}

	app.Commands = []cli.Command{
		{
			Name:      "avr",
			Usage:     "Renders all the plane paths found in an AVR file",
			Action:    parseAvr,
			ArgsUsage: "[outfile if not --stdout] input-file.txt [input-file.gz [input-file.bz2]]",
		},
		{
			Name:      "sbs",
			Aliases:   []string{"sbs1"},
			Usage:     "Renders all the plane paths found in an SBS file",
			Action:    parseSbs,
			ArgsUsage: "[outfile if not --stdout] input-file.txt [input-file.gz [input-file.bz2]]",
		},
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "v",
			Usage: "verbose debugging output",
		},
		cli.BoolFlag{
			Name:  "stdout",
			Usage: "Output to stdout instead of to a file (disables any other output)",
		},
	}

	tracker.MaxLocationHistory = -1
	//
	//f, err := os.Create("cpuprofile")
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "could not create CPU profile: %v\n", err)
	//	os.Exit(1)
	//}
	//if err := pprof.StartCPUProfile(f); err != nil {
	//	fmt.Fprintf(os.Stderr, "could not start CPU profile: %v\n", err)
	//	os.Exit(1)
	//}
	//defer pprof.StopCPUProfile()
	//

	if err := app.Run(os.Args); nil != err {
		fmt.Println(err)
	}
}

func produceOutput(c *cli.Context, newFrame frameFunc) (*producer, error) {
	stdOut := c.GlobalBool("stdout")

	p := producer{
		outFile:   "",
		dataFiles: []string{},
		verbose:   c.GlobalBool("v"),
		newFrame:  newFrame,
		logChan:   make(chan tracker.LogEvent, 50),
		input:     make(chan string, 50),
		errChan:   make(chan error, 50),
		out:       make(chan tracker.Event, 50),
	}

	if c.NArg() == 0 {
		return nil, fmt.Errorf("you need to specify some files")
	}

	if stdOut {
		p.debugOutput("Writing json output to stdout")
		p.dataFiles = c.Args()
	} else {
		p.outFile = c.Args().First()
		p.debugOutput("Writing json output to ", p.outFile)
		p.dataFiles = c.Args()[1:]
	}
	if 0 == len(p.dataFiles) {
		return nil, fmt.Errorf("you need to specify some files")
	}
	p.debugOutput("using ", len(p.dataFiles), " data files")

	go p.handleErrors()

	return &p, nil
}

func (p producer) debugOutput(v ...interface{}) {
	p.logChan <- tracker.LogEvent{
		When:    time.Now(),
		Level:   tracker.LogLevelDebug,
		Section: "Reader",
		Message: fmt.Sprint(v...),
	}
}

func (p *producer) Listen() chan tracker.Event {
	go p.readFiles()

	go func() {
		var lineCounter uint64
		for line := range p.input {
			_, _ = fmt.Fprintf(os.Stderr, "Processing line: %d\r", lineCounter)
			p.out <- p.newFrame(line)
			lineCounter++
			if 0 == lineCounter%10000 {
				_, _ = fmt.Fprintf(os.Stderr, "Processing line: %d\r", lineCounter)
			}
		}
		p.debugOutput("Done reading lines. Processed ", lineCounter, " lines")
		close(p.out)
		close(p.errChan)
	}()

	return p.out
}

func (p producer) Stop() {
	close(p.input)
	close(p.errChan)
}

func (p producer) handleErrors() {
	for err := range p.errChan {
		if nil != err {
			p.out <- tracker.NewLogEvent(tracker.LogLevelError, "Reader", fmt.Sprint(err))
		}
	}
}

func (p *producer) readFiles() {
	var err error
	var inFile *os.File
	var gzipFile *gzip.Reader
	for _, inFileName := range p.dataFiles {
		p.debugOutput("Loading lines from ", inFileName)
		inFile, err = os.Open(inFileName)
		if err != nil {
			p.errChan <- fmt.Errorf("failed to open file {%s}: %s", inFileName, err)
			continue
		}

		isGzip := strings.ToLower(inFileName[len(inFileName)-2:]) == "gz"
		isBzip2 := strings.ToLower(inFileName[len(inFileName)-3:]) == "bz2"
		p.debugOutput("Is Gzip?", isGzip, " Is Bzip2?", isBzip2)

		var scanner *bufio.Scanner
		if isGzip {
			gzipFile, err = gzip.NewReader(inFile)
			if nil != err {
				p.errChan <- err
				continue
			}
			scanner = bufio.NewScanner(gzipFile)
		} else if isBzip2 {
			bzip2File := bzip2.NewReader(inFile)
			scanner = bufio.NewScanner(bzip2File)
		} else {
			scanner = bufio.NewScanner(inFile)
		}
		for scanner.Scan() {
			p.input <- scanner.Text()
		}
		_ = inFile.Close()
	}
	close(p.input)
}

func writeResult(trk *tracker.Tracker, outFileName string) error {
	fc := geojson.NewFeatureCollection([]*geojson.Feature{})
	var coordCounter, planeCounter int
	var trackCounter int

	addFeature := func(coordinates geojson.Coordinates, p *tracker.Plane) {
		trackCounter++
		props := make(map[string]interface{})
		props["icao"] = p.IcaoIdentifierStr()
		props["trackNo"] = trackCounter
		if len(coordinates) > 1 {
			fc.AddFeatures(geojson.NewFeature(geojson.NewLineString(coordinates), props, fmt.Sprintf("%s_%d", p.IcaoIdentifierStr(), trackCounter)))
		}
	}

	trk.EachPlane(func(p *tracker.Plane) bool {
		var coords geojson.Coordinates
		if 0 == len(p.LocationHistory()) {
			return true
		}
		planeCounter++
		numLocations := len(p.LocationHistory())
		coords = make(geojson.Coordinates, 0, numLocations)
		for index, l := range p.LocationHistory() {
			if l.Lat() == 0.0 && l.Lon() == 0.0 {
				continue
			}

			coordCounter++
			coords = append(coords, geojson.Coordinate{geojson.CoordType(l.Lon()), geojson.CoordType(l.Lat())})

			if l.TrackFinished {
				addFeature(coords, p)
				coords = make(geojson.Coordinates, 0, numLocations-index)
			}
		}
		addFeature(coords, p)
		return true
	})
	_, _ = fmt.Fprintf(os.Stderr, "We have %d coords tracked over %d tracks from %d planes\n", coordCounter, trackCounter, planeCounter)

	jsonContent, err := json.Marshal(fc)
	//jsonContent, err := json.MarshalIndent(fc, "", "  ")
	if nil != err {
		panic(err)
	}
	if outFileName != "" {
		var f *os.File
		f, err = os.Create(outFileName)
		if nil == err {
			_, _ = f.Write(jsonContent)
			_ = f.Close()
			fmt.Println("Wrote content to file: " + outFileName)
			return nil
		}
	}
	// no writing to a file? output to screen!
	fmt.Println("\n" + string(jsonContent))
	return nil
}
