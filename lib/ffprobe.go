package lib

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

type ffprobeResult struct {
	Audio *ffprobeStreamInfo
	Video *ffprobeStreamInfo
}

type ffprobeStreamInfo struct {
	CodecType string `json:"codec_type"`
	StartPTS  int64  `json:"start_pts"`
	Duration  string `json:"duration"`
}
type fullFfprobeResult struct {
	Streams []*ffprobeStreamInfo   `json:"streams"`
	Format  map[string]interface{} `json:"format"`
}

type Ffprober interface {
	Run(io.Reader) (ffprobeResult, error)
}

func NewFfprober() Ffprober { return &ffprober{} }

type ffprober struct{}

func (f *ffprober) Run(in io.Reader) (ffprobeResult, error) {
	r := ffprobeResult{}
	bin := bufio.NewReader(in)
	tmpFile, err := ioutil.TempFile("", "ffprobe-tmp-src")
	if err != nil {
		return r, err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())
	bin.WriteTo(tmpFile)
	raw, err := runCommand(exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		tmpFile.Name()))
	if err != nil {
		return r, err
	}
	io.Copy(ioutil.Discard, in)
	fullOut := fullFfprobeResult{}
	if err := json.Unmarshal([]byte(raw), &fullOut); err != nil {
		return r, err
	}
	for _, s := range fullOut.Streams {
		local := s
		switch s.CodecType {
		case "video":
			r.Video = local
		case "audio":
			r.Audio = local
		}
	}

	return r, nil
}
