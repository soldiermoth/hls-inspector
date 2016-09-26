package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/grafov/m3u8"
	"github.com/soldiermoth/hls-inspector/lib"
)

var (
	m3u8URL           = flag.String("m3u8", "", "URL to m3u8 to watch & report on")
	variantSelection  = flag.Int("variant", -1, "Variant to pick")
	tsReportPath      = flag.String("tsreport", "tsreport", "Which tsreport bin to use")
	inspectionWorkers = flag.Int("inspection-workers", 10, "Number of Inspection Workers to Use")
)

func main() {
	flag.Parse()
	if *m3u8URL == "" {
		flag.Usage()
		log.Fatalln("Incorrect usage")
	}
	log.Printf("Inspecting %s", *m3u8URL)
	m, err := fetchM3U8(*m3u8URL)
	if err != nil {
		log.Fatalf("Error fetching m3u8: %+v", err)
	}
	segmentInspector := lib.NewSegmentInspector(*tsReportPath)
	mediaM3u8Inspector := lib.NewMediaM3U8Inspector(segmentInspector, *inspectionWorkers)
	segmentsInfo, err := mediaM3u8Inspector.InspectSegments(m)
	if err != nil {
		log.Fatalf("Error inspecting m3u8: %+v", err)
	}
	fmt.Println(segmentsInfo)
}

func fetchM3U8(url string) (*m3u8.MediaPlaylist, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	playlist, kind, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return nil, err
	}
	switch kind {
	case m3u8.MEDIA:
		return playlist.(*m3u8.MediaPlaylist), nil
	case m3u8.MASTER:
		master := playlist.(*m3u8.MasterPlaylist)
		if *variantSelection >= 0 {
			return fetchM3U8(master.Variants[*variantSelection].URI)
		}
		in := bufio.NewReader(os.Stdin)
		for {
			fmt.Println("Pick a Variant:")
			for i, m := range master.Variants {
				fmt.Printf("\t%d: %q\n", i, m.URI)
			}
			txt, _ := in.ReadString('\n')
			i, err := strconv.Atoi(strings.TrimSpace(txt))
			if err != nil || i < 0 || i >= len(master.Variants) {
				fmt.Printf("%q is not a valid selection", txt)
				continue
			}
			return fetchM3U8(master.Variants[i].URI)
		}

	}
	return nil, fmt.Errorf("Unknown Playlist Kind: %+v", kind)
}
