package lib

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/grafov/m3u8"
)

type MediaM3U8Inspector interface {
	InspectSegments(*url.URL, *m3u8.MediaPlaylist) (string, error)
}

func NewMediaM3U8Inspector(segmentInspector SegmentInspector, workerCount int) MediaM3U8Inspector {
	return &mediaM3U8Inspector{segmentInspector: segmentInspector, workerCount: workerCount}
}

type mediaM3U8Inspector struct {
	segmentInspector SegmentInspector
	workerCount      int
}

type segInspectJob struct {
	i   int
	seg *m3u8.MediaSegment
}
type segInspectResult struct {
	i    int
	err  error
	info SegmentInspection
}

var (
	tableLineTmpl = template.Must(template.New("table_line").Funcs(template.FuncMap{
		"minus": func(a, b time.Duration) time.Duration { return a - b },
	}).Parse(`
		{{- .i}}|
		{{- .manifestDur }}|{{ .ffprobe.Audio.Duration }}|{{ .ffprobe.Video.Duration }}|
		{{- .uniqueAudio }}|
		{{- .elapsed }}|{{ .elapsedAudio }}|{{ .elapsedVideo }}|{{ minus .elapsedAudio .elapsedVideo }}|
		{{- .uri }}|
		{{- .audioPTS }}|{{ .audioPTSDiff }}|{{ .cumulativeOverlap }}|
		{{- .videoPTS }}|{{ .videoPTSDiff }}|{{ .videoDTS }}|
		{{- .mediaInfo.WritingLibrary | printf "%.3s" }}|{{ .mediaInfo.DelayRelativeToVideo }}|{{ .mediaInfo.ColorPrimaries }}`))
)

func writeTableColumns(w io.Writer, cols ...string) {
	all := regexp.MustCompile(`.`)
	tab := func() { w.Write([]byte("\t")) }
	newline := func() { w.Write([]byte("\n")) }
	for i, col := range cols {
		colb := []byte(col)
		if i > 0 {
			tab()
		}
		w.Write(colb)
	}
	newline()
	for i, col := range cols {
		if i > 0 {
			tab()
		}
		w.Write(all.ReplaceAll([]byte(col), []byte("-")))
	}
	newline()
}

func (mmi *mediaM3U8Inspector) inspectAllSegments(mURL *url.URL, segs []*m3u8.MediaSegment) ([]*SegmentInspection, error) {
	infos := make([]*SegmentInspection, len(segs))
	jobs := make(chan segInspectJob, len(segs))
	results := make(chan segInspectResult, len(segs))

	for w := 1; w <= mmi.workerCount; w++ {
		go func(id int, jobs <-chan segInspectJob, results chan<- segInspectResult) {
			for j := range jobs {
				result := segInspectResult{i: j.i}
				if j.seg == nil {
					results <- result
					continue
				}
				log.Printf("Starting Segment #%d: %q", j.i, j.seg.URI)
				segURL, err := url.Parse(j.seg.URI)
				if err != nil {
					result.err = err
					results <- result
					continue
				}
				if mURL != nil && !segURL.IsAbs() {
					segURL = mURL.ResolveReference(segURL)
				}
				result.info, result.err = mmi.segmentInspector.Inspect(segURL.String())
				log.Printf("Finished Segment #%d: Info=%#v Error=%+v", j.i, result.info, result.err)
				results <- result
			}
		}(w, jobs, results)
	}

	for i, s := range segs {
		jobs <- segInspectJob{i: i, seg: s}
	}
	close(jobs)
	for range segs {
		result := <-results
		if result.err != nil {
			return nil, result.err
		}
		info := result.info
		infos[result.i] = &info
	}

	return infos, nil
}

func (mmi *mediaM3U8Inspector) InspectSegments(mURL *url.URL, m *m3u8.MediaPlaylist) (string, error) {
	sb := bytes.NewBuffer([]byte(""))
	tw := tabwriter.NewWriter(sb, 5, 4, 2, ' ', tabwriter.TabIndent)

	writeTableColumns(tw,
		"Segment #",
		"Duration", "Audio", "Video",
		"Unique Audio",
		"Elapsed", "Audio", "Video", "AV Diff",
		"URI",
		"Audio PTS", "Diff", "Cumulative Overlap",
		"Video PTS", "Diff", "DTS",
		"Writing Lib", "Delay to Video", "Color")
	infos, err := mmi.inspectAllSegments(mURL, m.Segments)
	if err != nil {
		return "", err
	}
	var prevInfo *SegmentInspection
	var elapsed, elapsedAudio, elapsedVideo, cumulativeOverlap time.Duration
	// elapsed, elapsedAudio, elapsedVideo, cumulativeOv := time.Duration(0), time.Duration(0), time.Duration(0)
	for i, s := range m.Segments {
		if s == nil {
			break
		}
		info := infos[i]
		if i-1 >= 0 {
			prevInfo = infos[i-1]
		}
		manifestDur := time.Duration(s.Duration * float64(time.Second))
		audioPTS, videoPTS, videoDTS, audioPTSDiff, videoPTSDiff := " ", " ", " ", " ", " "
		if info.Audio != nil {
			audioPTS = fmt.Sprintf("%d - %d", info.Audio.StartPTS, info.Audio.EndPTS)
			if prevInfo != nil {
				audioPTSDiff = strconv.FormatInt(info.Audio.StartPTS-prevInfo.Audio.EndPTS, 10)
			}
		}
		if info.Video != nil {
			videoPTS = fmt.Sprintf("%d - %d", info.Video.StartPTS, info.Video.EndPTS)
			if prevInfo != nil {
				videoPTSDiff = strconv.FormatInt(info.Video.StartPTS-prevInfo.Video.EndPTS, 10)
			}
			videoDTS = fmt.Sprintf("%d - %d", info.Video.StartDTS, info.Video.EndDTS)
			if videoPTS == videoDTS {
				videoDTS = " "
			}
		}
		if v := info.Ffprobe.Video; v != nil {
			if d, err := strconv.ParseFloat(v.Duration, 64); err == nil {
				elapsedVideo += time.Duration(d * float64(time.Second))
			}
		}
		uniqueAudio := time.Duration(0)
		if a := info.Ffprobe.Audio; a != nil {
			if d, err := strconv.ParseFloat(a.Duration, 64); err == nil {
				uniqueAudio = time.Duration(d * float64(time.Second))
				if prevInfo != nil && prevInfo.Audio != nil {
					if pd, err := strconv.ParseFloat(prevInfo.Ffprobe.Audio.Duration, 64); err == nil {
						overlap := time.Duration((float64(prevInfo.Audio.StartPTS)/90000.0 + pd - float64(info.Audio.StartPTS)/90000.0) * float64(time.Second))
						if overlap > 0 {
							cumulativeOverlap += overlap
							uniqueAudio -= cumulativeOverlap
						}
					}
				}
				elapsedAudio += uniqueAudio
			}
		}
		elapsed += manifestDur

		tbuf := &bytes.Buffer{}
		tableLineTmpl.Execute(tbuf, map[string]interface{}{
			"i":                 i,
			"manifestDur":       manifestDur,
			"uri":               urlPathLastN(s.URI, 25),
			"audioPTS":          audioPTS,
			"audioPTSDiff":      audioPTSDiff,
			"videoPTS":          videoPTS,
			"videoPTSDiff":      videoPTSDiff,
			"videoDTS":          videoDTS,
			"mediaInfo":         info.MediaInfo,
			"ffprobe":           info.Ffprobe,
			"elapsed":           elapsed,
			"elapsedVideo":      elapsedVideo,
			"elapsedAudio":      elapsedAudio,
			"uniqueAudio":       uniqueAudio,
			"cumulativeOverlap": cumulativeOverlap,
		})
		fmt.Fprintf(tw, "%s\n", strings.Replace(tbuf.String(), "|", "\t", -1))
	}
	tw.Flush()
	return sb.String(), nil
}
