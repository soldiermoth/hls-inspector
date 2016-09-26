package lib

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	mediainfoDelayRelativeToVideo = regexp.MustCompile(`^.*Delay relative to video\s+:\s+(-?\d+)ms.*$`)
	mediainfoWritingLibrary       = regexp.MustCompile(`^.*Writing library\s+:\s+(.*)$`)
	mediainfoColorPrimaries       = regexp.MustCompile(`^.*Color primaries\s+:\s+(.*)$`)
)

type mediainfoResult struct {
	DelayRelativeToVideo string
	WritingLibrary       string
	ColorPrimaries       string
}

type MediaInfoer interface {
	Run(io.Reader) (mediainfoResult, error)
}

func NewMediaInfoer() MediaInfoer { return &mediainfoer{} }

type mediainfoer struct{}

func (m *mediainfoer) Run(in io.Reader) (mediainfoResult, error) {
	bin := bufio.NewReader(in)
	r := mediainfoResult{}
	tmpFile, err := ioutil.TempFile("", "mediainfo-tmp-src")
	if err != nil {
		return r, err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())
	bin.WriteTo(tmpFile)
	mediainfo, err := runCommand(exec.Command("mediainfo", tmpFile.Name()))
	if err != nil {
		return r, err
	}
	for _, l := range strings.Split(mediainfo, "\n") {
		if s := mediainfoDelayRelativeToVideo.ReplaceAllString(l, "$1"); s != l {
			r.DelayRelativeToVideo = s
		}
		if s := mediainfoColorPrimaries.ReplaceAllString(l, "$1"); s != l {
			r.ColorPrimaries = s
		}
		if s := mediainfoWritingLibrary.ReplaceAllString(l, "$1"); s != l {
			r.WritingLibrary = s
		}
	}
	return r, nil
}
