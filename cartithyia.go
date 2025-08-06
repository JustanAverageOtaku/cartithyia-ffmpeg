package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
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

	switch path.Ext(*source) {
	case ".mp4":
	default:
		return errors.New("source file is not of a valid format")
	}

	if len(*destination) == 0 {
		return errors.New("destination cannot be empty")
	}

	switch path.Ext(*destination) {
	case ".jpg":
	default:
		return errors.New("destination file is not provided with a valid format")
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
	dpath := destination.Value.String()

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

	v1raw, err := os.ReadFile(v1path)
	if err != nil {
		return err
	}

	v2raw, err := os.ReadFile(v2path)
	if err != nil {
		return err
	}

	pipe1 := "1" + filepath.Base(v1path)
	if err := syscall.Mkfifo(pipe1, 0600); err != nil {
		return err
	}

	pipe2 := "2" + filepath.Base(v2path)
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
	var f3out, f3err bytes.Buffer
	cmd.Stdout = &f3out
	cmd.Stderr = &f3err

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error encoding merged stream. err:%s, f3err:%s", err, f3err.String())
	}

	if err := os.WriteFile(dpath, f3out.Bytes(), 0644); err != nil {
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
