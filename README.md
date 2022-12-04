# HLS Inspector

## Validate HLS streams
This script checks a HLS playlist for the following:

- Parse and validate m3u8 play list and find related variants. (play list should be parsable)
- Whether all .ts fragments can be downloaded for all quality levels
- Whether all .ts fragments have video frames
- Whether all .ts fragments has true encoding
- These are just the issues that we check for. There are other things that can go wrong with a HLS stream.

## Dependencies
* [Ffprobe](https://ffmpeg.org/download.html)
* [Mediainfo](https://mediaarea.net/en/MediaInfo)
* [tsreport](https://github.com/kynesim/tstools)

to install dependencies you can:
```
apt install ffmpeg -y
```

```
apt install mediainfo -y
```

```
apt install unzip -y
```

```
wget https://github.com/kynesim/tstools/archive/refs/heads/master.zip && \
    unzip master.zip && \
    cd  tstools-master/ && make && make install
```

## Install
1- download binary file and run it.
for linux:
```
wget https://github.com/Mehrdad-Dadkhah/hls-inspector/tree/develop/bin/hls-inspector
```

```
$hls-inspector -variant 0 -m3u8 'http://example.com/some.m3u8'
```
for help:
```
$hls-inspector
```

or
2- first clone hls-inspector and cd to hls-inspector then run:
```
go get
```

```
go run main.go -m3u8 'http://example.com/some.m3u8'
```

to check first variant of play list and pass variant question:
```
go run main.go -variant 0 -m3u8 'http://example.com/some.m3u8'
```

This will download, validate and print results for each .m3u8 and .ts URL.

The URL can point to either a playlist of .ts files, or to a master playlist of multiple quality playlists.

## Flags

- inspection-workers: Number of inspection workers to use. default is 10
- variant: Variant to pick. if -1 means wait to pick from user. default is -1
- tsreport: Which tsreport bin to use. path to `tsreport` bin. 
