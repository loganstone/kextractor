package main

import (
	"container/heap"
	"fmt"
	"log"
	"os"

	"github.com/loganstone/kpick/ask"
	"github.com/loganstone/kpick/conf"
	"github.com/loganstone/kpick/dir"
	"github.com/loganstone/kpick/file"
	"github.com/loganstone/kpick/profile"
	"github.com/loganstone/kpick/regex"
)

func showNumbers(foundFilesCnt int, scanErrorsCnt int, filesCntContainingKorean int) {
	fmt.Printf("[%d] scanning files\n", foundFilesCnt)
	fmt.Printf("[%d] error \n", scanErrorsCnt)
	fmt.Printf("[%d] success \n", foundFilesCnt-scanErrorsCnt)
	fmt.Printf("[%d] files containing korean\n", filesCntContainingKorean)
}

func main() {
	opts := conf.Opts()

	profile.CPU(opts.Cpuprofile)

	err := dir.Check(opts.DirToFind)
	if err != nil {
		log.Fatal(err)
	}

	skipPaths, err := regex.SkipPaths(opts.SkipPaths, ",", "|")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("find for files [*.%s] in [%s] directory\n", opts.FileExtToScan, opts.DirToFind)
	foundFiles, err := dir.Find(opts.DirToFind, opts.FileExtToScan, skipPaths)
	if err != nil {
		log.Fatal(err)
	}

	foundFilesCnt := len(foundFiles)
	if foundFilesCnt == 0 {
		fmt.Printf("[*.%s] file not found in [%s] directory\n", opts.FileExtToScan, opts.DirToFind)
		os.Exit(0)
	}

	if opts.Interactive {
		q := fmt.Sprintf("found files [%d]. do you want to scan it? (y/n): ", foundFilesCnt)
		ok, err := ask.Confirm(q, "y", "n")
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			os.Exit(0)
		}
	}

	match, ignore, err := regex.ForFileScan(conf.KoreanPatternForRegex, opts.IgnorePattern)
	if err != nil {
		log.Fatal(err)
	}

	filesContainingKorean := &file.SortedFiles{}
	heap.Init(filesContainingKorean)
	var scanErrorsCnt int
	beforeFn := func(filePath string) {
		if opts.Verbose {
			fmt.Printf("[%s] scanning for \"%s\"\n", filePath, match.String())
		}
	}
	afterFn := func(filePath string) {
		if opts.Verbose {
			fmt.Printf("[%s] scanning done\n", filePath)
		}
	}
	for _, paths := range file.Chunks(foundFiles) {
		for f := range file.ScanFiles(paths, match, ignore, beforeFn, afterFn) {
			if err := f.Error(); err != nil {
				scanErrorsCnt++
				if opts.Verbose || opts.ErrorOnly {
					fmt.Printf("[%s] scanning error - %s\n", f.Path(), err)
				}
				continue
			}

			if len(f.FoundLines()) > 0 {
				heap.Push(filesContainingKorean, f)
			}
		}
	}

	if !opts.ErrorOnly {
		file.PrintFiles(filesContainingKorean)
	}

	showNumbers(foundFilesCnt, scanErrorsCnt, filesContainingKorean.Len())

	profile.Mem(opts.Memprofile)
}
