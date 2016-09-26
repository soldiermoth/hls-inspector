package lib

import (
	"fmt"
	"io"
	"net/http"
)

type StreamInfo struct {
	StartPTS, EndPTS int64
	StartDTS, EndDTS int64
	StartCC, EndCC   int
}

type SegmentInspection struct {
	Audio     *StreamInfo
	Video     *StreamInfo
	MediaInfo mediainfoResult
	Ffprobe   ffprobeResult
}

type SegmentInspector interface {
	Inspect(string) (SegmentInspection, error)
}

func NewSegmentInspector(tsreport string) SegmentInspector {
	return &segmentInspector{
		tsReporter:  NewTSReporter(tsreport),
		mediainfoer: NewMediaInfoer(),
		ffprober:    NewFfprober(),
	}
}

type segmentInspector struct {
	tsReporter  TSReporter
	mediainfoer MediaInfoer
	ffprober    Ffprober
}

func (si *segmentInspector) Inspect(uri string) (SegmentInspection, error) {
	info := SegmentInspection{}
	resp, err := http.Get(uri)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("Error Fetching Segment %q, httpStatusCode=%d", uri, resp.StatusCode)
	}
	tsreportR, tsreportW := io.Pipe()
	mediainfoR, mediainfoW := io.Pipe()
	ffprobeR, ffprobeW := io.Pipe()
	tsreportchan := make(chan tsReportResult, 1)
	mediainfochan := make(chan mediainfoResult, 1)
	ffprobechan := make(chan ffprobeResult, 1)
	defer close(tsreportchan)
	defer close(mediainfochan)
	defer close(ffprobechan)

	go func() {
		r, err := si.tsReporter.Run(tsreportR)
		if err != nil {
			fmt.Printf("Error generating ts report: %+v\n", err)
		}
		tsreportchan <- r
	}()
	go func() {
		r, err := si.mediainfoer.Run(mediainfoR)
		if err != nil {
			fmt.Printf("Error generating mediainfo: %+v\n", err)
		}
		mediainfochan <- r
	}()

	go func() {
		r, err := si.ffprober.Run(ffprobeR)
		if err != nil {
			fmt.Printf("Error running ffprobe: %+v\n", err)
		}
		ffprobechan <- r
	}()
	go func() {
		defer tsreportW.Close()
		defer mediainfoW.Close()
		defer ffprobeW.Close()
		mw := io.MultiWriter(tsreportW, mediainfoW, ffprobeW)
		if _, err := io.Copy(mw, resp.Body); err != nil {
			fmt.Printf("Error copying in to multiwriter: %+v\n", err)
		}
	}()

	tsreport, mediainfo, ffprobe := <-tsreportchan, <-mediainfochan, <-ffprobechan

	info.Audio = tsreport.audio
	info.Video = tsreport.video
	info.MediaInfo = mediainfo
	info.Ffprobe = ffprobe
	return info, nil
}
