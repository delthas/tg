package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func fatal(f string, args ...interface{}) {
	a := make([]interface{}, len(args)+1)
	a[0] = os.Args[0]
	copy(a[1:], args)
	_, _ = fmt.Fprintf(os.Stderr, "%s: "+f+"\n", a...)
	os.Exit(1)
}

// bufio.ScanLines but split at \r, \n, or \r\n, and keep line endings
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	n := bytes.IndexByte(data, '\n')
	var r int
	if n == -1 {
		r = bytes.IndexByte(data, '\r')
	} else {
		r = bytes.IndexByte(data[:n], '\r')
	}
	if r != -1 {
		if r == len(data) - 1 {
			if atEOF {
				return len(data), data, nil
			}
			// request more data to check if next char is \n
			return 0, nil, nil
		}
		if data[r + 1] == '\n' {
			r++
		}
		return r + 1, data[:r+1], nil
	}
	if n != -1 {
		return n + 1, data[:n+1], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	// request more data
	return 0, nil, nil
}

var patterns = make(map[string][]*regexp.Regexp, len(os.Args) / 2)

func match(line string, stderr bool) bool {
	for _, r := range patterns["A"] {
		if !r.MatchString(line) {
			return false
		}
	}
	for _, r := range patterns["a"] {
		if r.MatchString(line) {
			return false
		}
	}
	var includePatterns []*regexp.Regexp
	var excludePatterns []*regexp.Regexp
	if stderr {
		includePatterns = patterns["E"]
		excludePatterns = patterns["e"]
	} else {
		includePatterns = patterns["O"]
		excludePatterns = patterns["o"]
	}
	for _, r := range includePatterns {
		if !r.MatchString(line) {
			return false
		}
	}
	for _, r := range excludePatterns {
		if r.MatchString(line) {
			return false
		}
	}
	return true
}

var loaded = make(map[string]struct{}, 4)

func load(path string) {
	if _, ok := loaded[path]; ok {
		return
	}
	loaded[path] = struct{}{}
	f, err := os.Open(path)
	if err != nil {
		fatal("config file %s could not be opened: %s", path, err.Error())
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		if len(l) == 0 {
			continue
		}
		if !strings.HasPrefix(l, "-") {
			fatal("invalid line in config file %s: line doesnt start with `-`: %s", path, l)
		}
		i := strings.IndexRune(l, ' ')
		if i == -1 {
			fatal("invalid line in config file %s: line doesnt have pattern: %s", path, l)
		}
		key := l[1:i]
		value := l[i+1:]
		switch key {
		case "o", "e", "a", "O", "E", "A":
			r, err := regexp.Compile(value)
			if err != nil {
				fatal("invalid line in config file %s: pattern not recognized: %s: %s", path, err.Error(), value)
			}
			patterns[key] = append(patterns[key], r)
		case "c":
			if !filepath.IsAbs(value) {
				p, err := filepath.Abs(path)
				if err != nil {
					fatal("could not get absolute path of file %s: %s", path, err.Error())
				}
				value = filepath.Join(filepath.Dir(p), value)
			}
			load(value)
		case "-":
			continue
		default:
			fatal("invalid line in config file %s: line has unknown rule type: %s", path, l)
		}
	}
}

func main() {
	flagType := ""
	command := -1
	for i, v := range os.Args[1:] {
		if flagType == "" {
			if strings.HasPrefix(v, "-") {
				switch v[1:] {
				case "o", "e", "a", "O", "E", "A", "c":
					flagType = v[1:]
				case "-":
					command = i + 1
					break
				default:
					fatal("flag not recognized: %s", v)
				}
			} else {
				command = i
				break
			}
		} else {
			if flagType == "c" {
				load(v)
			} else {
				r, err := regexp.Compile(v)
				if err != nil {
					fatal("pattern not recognized: %s: %s", err.Error(), v)
				}
				patterns[flagType] = append(patterns[flagType], r)
			}
			flagType = ""
		}
	}
	command++ // for loop started at index 1
	if flagType != "" {
		fatal("not enough arguments, value for -%s was not specified", flagType)
	}
	if command == 0 || command >= len(os.Args) {
		fatal("not enough arguments, a command must be specified")
	}
	cmd := exec.Command(os.Args[command], os.Args[command+1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fatal("failed creating stdout pipe: %s", err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fatal("failed creating stderr pipe: %s", err.Error())
	}
	err = cmd.Start()
	if err != nil {
		fatal("failed starting %s: %s", cmd.Path, err.Error())
	}
	rout := bufio.NewScanner(stdout)
	rerr := bufio.NewScanner(stderr)
	scanner := func(sc *bufio.Scanner, stderr bool) {
		sc.Split(scanLines)
		for sc.Scan() {
			line := sc.Text()
			i := len(line)
			if len(line) >= 1 {
				switch line[len(line) - 1] {
				case '\r':
					i--
				case '\n':
					i--
					if len(line) >= 2 && line[len(line) - 2] == '\r' {
						i--
					}
				}
			}
			if !match(line[:i], stderr) {
				continue
			}
			if stderr {
				_, _ = fmt.Fprint(os.Stderr, line)
			} else {
				_, _ = fmt.Fprint(os.Stdout, line)
			}
		}
	}
	scanner(rout, false)
	scanner(rerr, true)
	err = cmd.Wait()
	if err != nil {
		if ee := err.(*exec.ExitError); ee != nil {
			code := ee.ExitCode()
			if code >= 0 {
				os.Exit(code)
			}
			fatal("process did not exit normally: %s", ee.Error())
		}
		fatal("failed running process: %s", err.Error())
	}
}
