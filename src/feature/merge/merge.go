package merge

import (
	"bytes"
	"cartithyia/src/feature"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/google/uuid"
)

const (
	FeatureName = "merge"
)

type (
	impl struct {
		flagSet *flag.FlagSet
	}

	MergeArgs struct {
		SourceV1    string
		SourceV2    string
		Destination string
	}
)

var (
	supportedVideoFormats = map[string]struct{}{
		".mp4": {},
	}
)

func (i *impl) Execute(args []string) error {
	margs, err := parseMergeArgs(i.flagSet, args)
	if err != nil {
		return err
	}

	return mergeVideos(margs)
}

func parseMergeArgs(fset *flag.FlagSet, args []string) (MergeArgs, error) {
	video1 := fset.String("v1", "", "v1")
	video2 := fset.String("v2", "", "v2")
	destination := fset.String("destination", "", "destination")

	if err := fset.Parse(args); err != nil {
		return MergeArgs{}, err
	}

	if len(*video1) == 0 {
		return MergeArgs{}, feature.ErrEmptySource
	}

	ext := path.Ext(*video1)
	if _, ok := supportedVideoFormats[ext]; !ok {
		return MergeArgs{}, fmt.Errorf("supported formats: %+v, got:%s", supportedVideoFormats, ext)
	}

	if len(*video2) == 0 {
		return MergeArgs{}, feature.ErrEmptySource
	}

	ext = path.Ext(*video2)
	if _, ok := supportedVideoFormats[ext]; !ok {
		return MergeArgs{}, fmt.Errorf("supported formats: %+v, got:%s", supportedVideoFormats, ext)
	}

	if len(*destination) == 0 {
		return MergeArgs{}, feature.ErrEmptyDestination
	}

	ext = path.Ext(*destination)
	if _, ok := supportedVideoFormats[ext]; !ok {
		return MergeArgs{}, fmt.Errorf("supported formats: %+v, got:%s", supportedVideoFormats, ext)
	}

	return MergeArgs{
		SourceV1:    *video1,
		SourceV2:    *video2,
		Destination: *destination,
	}, nil
}

func mergeVideos(fargs MergeArgs) error {
	v1raw, err := os.ReadFile(fargs.SourceV1)
	if err != nil {
		return err
	}

	v2raw, err := os.ReadFile(fargs.SourceV2)
	if err != nil {
		return err
	}

	pipe1 := uuid.NewString()
	if err := syscall.Mkfifo(pipe1, 0600); err != nil {
		return err
	}

	pipe2 := uuid.NewString()
	if err := syscall.Mkfifo(pipe2, 0600); err != nil {
		return err
	}

	go func() {
		f, _ := os.OpenFile(pipe1, os.O_WRONLY, os.ModeNamedPipe)
		defer f.Close()
		f.Write(v1raw)
	}()

	go func() {
		f, _ := os.OpenFile(pipe2, os.O_WRONLY, os.ModeNamedPipe)
		defer f.Close()
		f.Write(v2raw)
	}()

	cmd := exec.Command(
		"ffmpeg",
		"-i", pipe1,
		"-i", pipe2,
		"-filter_complex", "[0:v][1:v]concat=n=2:v=1:a=0[outv]",
		"-map", "[outv]",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov",
		"pipe:1",
	)
	var fout, ferr bytes.Buffer
	cmd.Stdout = &fout
	cmd.Stderr = &ferr

	if err := cmd.Run(); err != nil {
		return errors.Join(err, errors.New(ferr.String()))
	}

	if err := os.WriteFile(fargs.Destination, fout.Bytes(), 0644); err != nil {
		return err
	}

	if err := os.Remove(pipe1); err != nil {
		fmt.Printf("remove pipe1: %s", err)
	}

	if err := os.Remove(pipe2); err != nil {
		fmt.Printf("remove pipe2: %s", err)
	}

	return nil
}

func NewFeature(fset *flag.FlagSet) feature.Feature {
	return &impl{
		flagSet: fset,
	}
}
