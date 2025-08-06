package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
)

func main() {
	frame := flag.NewFlagSet("frame", flag.ExitOnError)
	merge := flag.NewFlagSet("merge", flag.ExitOnError)

	if len(os.Args) < 2 {
		panic("not enough arguments")
	}

	switch os.Args[1] {
	case frame.Name():
		if err := parseFrameArgs(frame, os.Args[2:]); err != nil {
			panic(err)
		}
		if err := splitFrame(frame); err != nil {
			panic(err)
		}
	case merge.Name():
		if err := parseMergeArgs(merge, os.Args[2:]); err != nil {
			panic(err)
		}
		if err := mergeVideos(merge); err != nil {
			panic(err)
		}
	default:
		fmt.Printf("O' Righteous One, my abilities are limited to: %+v", []flag.FlagSet{*frame, *merge})
		os.Exit(1)
	}
}

func parseFrameArgs(fset *flag.FlagSet, args []string) error {
	source := fset.String("source", "", "source")
	destination := fset.String("destination", "", "destination")

	if err := fset.Parse(args); err != nil {
		return err
	}

	if len(*source) == 0 {
		return errors.New("source cannot be empty")
	}

	if len(*destination) == 0 {
		return errors.New("destination cannot be empty")
	}

	return nil
}

func splitFrame(fset *flag.FlagSet) error {
	source := fset.Lookup("source")
	if source == nil {
		return errors.New("source is empty")
	}

	destination := fset.Lookup("destination")
	if destination == nil {
		return errors.New("source is empty")
	}

	spath := source.Value.String()
	switch path.Ext(spath) {
	case ".mp4":
	default:
		return errors.New("source file is not of a valid format")
	}

	dpath := destination.Value.String()
	switch path.Ext(dpath) {
	case ".jpg":
	default:
		return errors.New("destination file is not provided with a valid format")
	}

	fInfo, err := os.Stat(spath)
	if err != nil {
		return err
	}

	if fInfo.IsDir() {
		return errors.New("source cannot be a directory")
	}

	file, err := os.Open(spath)
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

	var frame, eBuffer bytes.Buffer
	cmd.Stdout = &frame
	cmd.Stderr = &eBuffer

	if err := cmd.Run(); err != nil {
		return err
	}

	if err := os.WriteFile(dpath, frame.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Printf("Frame Split-ed? Size:%d", frame.Len())

	return nil
}

func parseMergeArgs(fset *flag.FlagSet, args []string) error {
	video1 := fset.String("v1", "", "v1")
	video2 := fset.String("v2", "", "v2")
	destination := fset.String("destination", "", "destination")

	if err := fset.Parse(args); err != nil {
		return err
	}

	if len(*video1) == 0 {
		return errors.New("source cannot be empty")
	}

	switch path.Ext(*video1) {
	case ".mp4":
	default:
		return errors.New("v1 is not of a valid format")
	}

	if len(*video2) == 0 {
		return errors.New("destination cannot be empty")
	}

	switch path.Ext(*video2) {
	case ".mp4":
	default:
		return errors.New("v2 is not of a valid format")
	}

	if len(*destination) == 0 {
		return errors.New("destination cannot be empty")
	}

	switch path.Ext(*destination) {
	case ".mp4", ".mkv":
	default:
		return errors.New("destination is not of a valid format")
	}

	return nil
}

func mergeVideos(fset *flag.FlagSet) error {
	video1 := fset.Lookup("v1")
	if video1 == nil {
		return errors.New("v1 is empty")
	}

	video2 := fset.Lookup("v2")
	if video2 == nil {
		return errors.New("v2 is empty")
	}

	destination := fset.Lookup("destination")
	if destination == nil {
		return errors.New("destination is empty")
	}

	v1path := video1.Value.String()
	v2path := video2.Value.String()
	dpath := destination.Value.String()

	f1, err := os.Open(v1path)
	if err != nil {
		return err
	}
	defer f1.Close()

	cmd1 := exec.Command(
		"ffmpeg",
		"-f", "mp4", "-i", "pipe:0",
		"-f", "rawvideo", "-pix_fmt", "yuv420p",
		"pipe:1",
	)
	var f1out, f1err bytes.Buffer
	cmd1.Stdin = f1
	cmd1.Stdout = &f1out
	cmd1.Stderr = &f1err

	if err := cmd1.Run(); err != nil {
		return fmt.Errorf("error decoding %s: %s, f1err:%s", v1path, err, f1err.String())
	}

	fmt.Printf("f1out: %d\n", f1out.Len())

	f2, err := os.Open(v2path)
	if err != nil {
		return err
	}
	defer f2.Close()

	cmd2 := exec.Command(
		"ffmpeg",
		"-f", "mp4", "-i", "pipe:0",
		"-f", "rawvideo", "-pix_fmt", "yuv420p",
		"pipe:1",
	)
	var f2out, f2err bytes.Buffer
	cmd2.Stdin = f2
	cmd2.Stdout = &f2out
	cmd2.Stderr = &f2err

	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("error decoding %s: %s, f2err:%s", v2path, err, f2err.String())
	}

	fmt.Printf("f2out: %d\n", f2out.Len())

	var merged bytes.Buffer
	merged.Write(f1out.Bytes())
	//merged.Write(f2out.Bytes())

	cmd3 := exec.Command(
		"ffmpeg",
		"-f", "rawvideo",
		"-pix_fmt", "yuv420p",
		"-s", "1920x1080",
		"-r", "30",
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov",
		"pipe:1",
	)
	var f3out, f3err bytes.Buffer
	cmd3.Stdin = &merged
	cmd3.Stdout = &f3out
	cmd3.Stderr = &f3err

	if err := cmd3.Run(); err != nil {
		return fmt.Errorf("error encoding merged stream. err:%s, f3err:%s", err, f3err.String())
	}

	if err := os.WriteFile(dpath, f3out.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}
