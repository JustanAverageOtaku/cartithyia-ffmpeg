package frame

import (
	"bytes"
	"cartithyia/src/feature"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
)

const (
	FeatureName = "frame"
)

type (
	impl struct {
		flagSet *flag.FlagSet
	}

	FrameArgs struct {
		Source      string
		Destination string
	}
)

var (
	supportedVideoFormats = map[string]struct{}{
		".mp4": {},
	}

	supportedImageFormats = map[string]struct{}{
		".jpg": {},
	}
)

func (i *impl) Execute(args []string) error {
	fargs, err := parseFrameArgs(i.flagSet, args)
	if err != nil {
		return err
	}

	return splitFrame(fargs)
}

func parseFrameArgs(fset *flag.FlagSet, args []string) (FrameArgs, error) {
	source := fset.String("source", "", "source")
	destination := fset.String("destination", "", "destination")

	if err := fset.Parse(args); err != nil {
		return FrameArgs{}, err
	}

	if len(*source) == 0 {
		return FrameArgs{}, feature.ErrEmptySource
	}

	ext := path.Ext(*source)
	if _, ok := supportedVideoFormats[ext]; !ok {
		return FrameArgs{}, fmt.Errorf("supported video formats: %+v, got:%s", supportedVideoFormats, ext)
	}

	if len(*destination) == 0 {
		return FrameArgs{}, feature.ErrEmptyDestination
	}

	ext = path.Ext(*destination)
	if _, ok := supportedImageFormats[ext]; !ok {
		return FrameArgs{}, fmt.Errorf("supported image formats: %+v, got:%s", supportedImageFormats, ext)
	}

	return FrameArgs{
		Source:      *source,
		Destination: *destination,
	}, nil
}

func splitFrame(fargs FrameArgs) error {
	fInfo, err := os.Stat(fargs.Source)
	if err != nil {
		return err
	}

	if fInfo.IsDir() {
		return feature.ErrNotFile
	}

	file, err := os.Open(fargs.Source)
	if err != nil {
		return err
	}
	defer file.Close()

	cmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0",
		"-vframes", "1",
		"-f", "image2",
		"-vcodec", "mjpeg",
		"pipe:1",
	)
	cmd.Stdin = file

	var frame, ferr bytes.Buffer
	cmd.Stdout = &frame
	cmd.Stderr = &ferr

	if err := cmd.Run(); err != nil {
		return errors.Join(err, errors.New(ferr.String()))
	}

	if err := os.WriteFile(fargs.Destination, frame.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}

func NewFeature(fset *flag.FlagSet) feature.Feature {
	return &impl{
		flagSet: fset,
	}
}
