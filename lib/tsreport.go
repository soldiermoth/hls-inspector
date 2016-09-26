package lib

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	tsreportPTS = regexp.MustCompile(`^.*PTS\s+(\d+).*$`)
	tsreportDTS = regexp.MustCompile(`^.*DTS\s+(\d+).*$`)
)

type tsReportResult struct {
	audio *StreamInfo
	video *StreamInfo
}

type TSReporter interface {
	Run(io.Reader) (tsReportResult, error)
}

func NewTSReporter(path string) TSReporter { return &tsreporter{path: path} }

type tsreporter struct {
	path string
}

func (t *tsreporter) buildCommand(args ...string) *exec.Cmd {
	pathSplit := strings.Split(t.path, " ")
	if len(pathSplit) == 1 {
		return exec.Command(pathSplit[0], args...)
	}
	args = append(pathSplit[1:], args...)
	return exec.Command(pathSplit[0], args...)
}

func (t *tsreporter) Run(in io.Reader) (tsReportResult, error) {
	result := tsReportResult{}
	tsreport, err := runCommandWithStdin(in, t.buildCommand("-v", "-stdin", "-timing"))
	if err != nil {
		return result, fmt.Errorf("Error Inspecting Segment error=%+v", err)
	}
	tsreportlines := strings.Split(tsreport, "\n")
	for i, l := range tsreportlines {
		if strings.Contains(l, "Stream ID:") && i+4 < len(tsreportlines) {
			ptsraw := tsreportPTS.ReplaceAllString(tsreportlines[i+4], "$1")
			pts, err := strconv.ParseInt(ptsraw, 10, 64)
			if err != nil {
				return result, fmt.Errorf("Error parsing PTS from line %q", tsreportlines[i+4])
			}
			dts := pts
			if i+4 < len(tsreportlines) && strings.Contains(tsreportlines[i+5], "DTS") {
				dtsraw := tsreportDTS.ReplaceAllString(tsreportlines[i+5], "$1")
				dts, err = strconv.ParseInt(dtsraw, 10, 64)
				if err != nil {
					return result, fmt.Errorf("Error parsing DTS from line %q", tsreportlines[i+5])
				}
			}
			var stream *StreamInfo
			if strings.Contains(l, "Audio") {
				if result.audio == nil {
					result.audio = &StreamInfo{StartPTS: pts, StartDTS: dts}
				}
				stream = result.audio
			} else if strings.Contains(l, "Video") {
				if result.video == nil {
					result.video = &StreamInfo{StartPTS: pts, StartDTS: dts}
				}
				stream = result.video
			}
			stream.EndPTS = pts
			stream.EndDTS = dts
		}
	}
	return result, nil
}
