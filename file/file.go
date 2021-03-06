package file

import (
	"bufio"
	"container/heap"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"sync"
)

// BeforeScanFunc .
type BeforeScanFunc func(path string)

// AfterScanFunc .
type AfterScanFunc func(path string)

// File finds and stores the line matching the specified regular expression.
type File struct {
	path         string
	matchRegex   *regexp.Regexp
	ignoreRegex  *regexp.Regexp
	matchedLines map[int][]byte
	scanError    error
}

// Scan checks the contents of the file line by line to see
// if it matches the regular expression.
// When it finds a line that matches the regular expression,
// it stores the contents of the line with the line number.
func (f *File) Scan() {
	if f.matchRegex == nil {
		return
	}

	file, err := os.Open(f.path)
	if err != nil {
		f.scanError = err
		return
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	line := []byte{}
	var lineNumber int

	for {
		chunk, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				f.scanError = err
			}
			break
		}

		line = append(line, chunk...)
		if isPrefix {
			// NOTE(hs.lee): 줄 읽기가 다 끝나지 않았음. line 유지
			continue
		}

		// NOTE(hs.lee): 줄 읽기가 끝남
		lineNumber++
		if f.ignoreRegex != nil && f.ignoreRegex.Match(line) {
			line = []byte{}
			continue
		}

		if f.matchRegex.Match(line) {
			f.matchedLines[lineNumber] = line
		}

		line = []byte{}
	}
}

// Path returns a file path.
func (f *File) Path() string {
	return f.path
}

// Error returns an error scanned file.
func (f *File) Error() error {
	return f.scanError
}

// MatchedLines returns the result of Scan.
func (f *File) MatchedLines() map[int][]byte {
	return f.matchedLines
}

func (f *File) printMatchedLines() {
	lineNumbers := make([]int, len(f.matchedLines))
	var i int
	for lineNumber := range f.matchedLines {
		lineNumbers[i] = lineNumber
		i++
	}

	sort.Ints(lineNumbers)
	for _, lineNumber := range lineNumbers {
		lineText, _ := f.matchedLines[lineNumber]
		fmt.Printf("%d: %s\n", lineNumber, lineText)
	}
}

// Heap is a data type for sorting the file list in ascending order by name.
type Heap []*File

func (h Heap) Len() int {
	return len(h)
}

func (h Heap) Less(i, j int) bool {
	return h[i].path < h[j].path
}

func (h Heap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push .
func (h *Heap) Push(x interface{}) {
	*h = append(*h, x.(*File))
}

// Pop .
func (h *Heap) Pop() interface{} {
	old := *h
	n := len(old)
	element := old[n-1]
	*h = old[0 : n-1]
	return element
}

// Print is prints data of Heap.
func (h Heap) Print() {
	for h.Len() > 0 {
		f, ok := heap.Pop(&h).(*File)
		if ok {
			fmt.Println(f.Path())
			f.printMatchedLines()
		}
	}
}

// ScanFiles .
func ScanFiles(filePaths []string, m, ig *regexp.Regexp,
	beforeFn BeforeScanFunc, afterFn AfterScanFunc) <-chan *File {
	cp := make(chan *File)

	var wg sync.WaitGroup
	wg.Add(len(filePaths))

	for _, filePath := range filePaths {
		go func(filePath string) {
			defer wg.Done()
			beforeFn(filePath)
			f := &File{filePath, m, ig, map[int][]byte{}, nil}
			f.Scan()
			afterFn(filePath)
			cp <- f
		}(filePath)
	}

	go func() {
		wg.Wait()
		close(cp)
	}()
	return cp
}
