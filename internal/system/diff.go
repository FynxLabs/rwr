package system

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/pmezard/go-difflib/difflib"
	"golang.org/x/term"
)

type DiffOption struct {
	NoHeader         bool
	TabSize          int
	SeparatorSymbol  string
	SeparatorWidth   int
	SpaceSizeAfterLn int
}

func WriteUnifiedDiff(w io.Writer, diff difflib.UnifiedDiff, opt DiffOption) error {
	lnSpaceSize := countDigits(max(len(diff.A), len(diff.B)))

	width, _, err := terminalShape()
	if err != nil {
		// Fallback to a default width if unable to determine terminal dimensions
		width = 80
	}

	buf := bufio.NewWriter(w)
	defer func(buf *bufio.Writer) {
		err := buf.Flush()
		if err != nil {
			log.Fatalf("error flushing buffer: %v", err)
		}
	}(buf)

	groupedOpcodes := difflib.NewMatcher(diff.A, diff.B).GetGroupedOpCodes(diff.Context)
	for i, opcodes := range groupedOpcodes {
		if i == 0 && !opt.NoHeader {
			_, err := buf.WriteString(unifiedHeader(diff.FromFile, diff.ToFile))
			if err != nil {
				return err
			}
		}
		for _, c := range opcodes {
			i1, i2, j1, j2 := c.I1, c.I2, c.J1, c.J2
			if c.Tag == 'e' {
				for ln, line := range diff.A[i1:i2] {
					texts := splitText(line, width-2-lnSpaceSize*2-1, opt.TabSize)
					_, err := buf.WriteString(
						fmt.Sprintf(
							"%*d %*d%s%s\n",
							lnSpaceSize, i1+ln+1,
							lnSpaceSize, j1+ln+1,
							strings.Repeat(" ", opt.SpaceSizeAfterLn),
							texts[0],
						),
					)
					if err != nil {
						log.Fatalf("error writing to buffer: %v", err)
					}
					for i := 1; i < len(texts); i++ {
						_, err := buf.WriteString(
							fmt.Sprintf(
								"%s %s\n",
								strings.Repeat(" ", opt.SpaceSizeAfterLn+lnSpaceSize*2),
								texts[i],
							),
						)
						if err != nil {
							log.Fatalf("error writing to buffer: %v", err)
						}
					}
				}
			}
			if c.Tag == 'r' || c.Tag == 'd' {
				for ln, line := range diff.A[i1:i2] {
					texts := splitText(line, width-2-lnSpaceSize*2-1, opt.TabSize)
					_, err := buf.WriteString(
						fmt.Sprintf(
							"%*d %s%s\n",
							lnSpaceSize, i1+ln+1,
							strings.Repeat(" ", lnSpaceSize+opt.SpaceSizeAfterLn),
							texts[0],
						),
					)
					if err != nil {
						log.Fatalf("error writing to buffer: %v", err)
					}
					for i := 1; i < len(texts); i++ {
						_, err := buf.WriteString(
							fmt.Sprintf(
								"%s %s\n",
								strings.Repeat(" ", opt.SpaceSizeAfterLn+lnSpaceSize*2),
								texts[i],
							),
						)
						if err != nil {
							log.Fatalf("error writing to buffer: %v", err)
						}
					}
				}
			}
			if c.Tag == 'r' || c.Tag == 'i' {
				for ln, line := range diff.B[j1:j2] {
					texts := splitText(line, width-2-lnSpaceSize*2-1, opt.TabSize)
					_, err := buf.WriteString(
						fmt.Sprintf(
							" %*d%s%s\n",
							lnSpaceSize*2, j1+ln+1,
							strings.Repeat(" ", opt.SpaceSizeAfterLn),
							texts[0],
						),
					)
					if err != nil {
						log.Fatalf("error writing to buffer: %v", err)
					}
					for i := 1; i < len(texts); i++ {
						_, err := buf.WriteString(
							fmt.Sprintf(
								"%s %s\n",
								strings.Repeat(" ", opt.SpaceSizeAfterLn+lnSpaceSize*2),
								texts[i],
							),
						)
						if err != nil {
							log.Fatalf("error writing to buffer: %v", err)
						}
					}
				}
			}
		}
		if i != len(groupedOpcodes)-1 {
			_, err := buf.WriteString(fmt.Sprintf("%s\n", strings.Repeat(opt.SeparatorSymbol, opt.SeparatorWidth)))
			if err != nil {
				log.Fatalf("error writing to buffer: %v", err)
			}
		}
	}
	return nil
}

func unifiedHeader(org, new string) string {
	return fmt.Sprintf(
		"--- %s\n+++ %s\n",
		org,
		new,
	)
}

func countDigits(v int) int {
	var cnt int
	for v != 0 {
		v /= 10
		cnt++
	}
	return cnt
}

func formatTextLine(text string, tabSize int) string {
	text = strings.TrimSuffix(text, "\n")
	text = strings.ReplaceAll(text, "\t", strings.Repeat(" ", tabSize))
	return text
}

func splitText(text string, length, tabSize int) []string {
	text = formatTextLine(text, tabSize)
	if len(text) < length {
		return []string{text}
	}
	var res []string
	for i := 0; i < len(text); i += length {
		if i+length < len(text) {
			res = append(res, text[i:(i+length)])
		} else {
			res = append(res, text[i:])
		}
	}
	return res
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func terminalShape() (int, int, error) {
	// Get the terminal dimensions using the term package
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, 0, err
	}

	// Assume the terminal is half the width of a 1080p screen
	if width > 960 {
		width = 960
	}

	return width, height, nil
}

func ShowDiff(source, target string) error {
	sourceContent, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	targetContent, err := os.ReadFile(target)
	if err != nil {
		return err
	}

	// Create a new instance of the differ
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(sourceContent)),
		B:        difflib.SplitLines(string(targetContent)),
		FromFile: filepath.Base(source),
		ToFile:   filepath.Base(target),
		Context:  3,
	}

	// Configure the diff options
	opt := DiffOption{
		NoHeader:         false,
		TabSize:          4,
		SeparatorSymbol:  "-",
		SeparatorWidth:   80,
		SpaceSizeAfterLn: 2,
	}

	// Write the diff output
	err = WriteUnifiedDiff(os.Stdout, diff, opt)
	if err != nil {
		return err
	}

	return nil
}
