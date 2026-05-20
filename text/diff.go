package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ChangeType int

const (
	Unchanged ChangeType = iota
	Modified
	New
)

type FileChange struct {
	Lines map[int]ChangeType
}

func getGitDiff(commit, gitRoot string) map[string]FileChange {
	diffMap := make(map[string]FileChange)

	cmd := exec.Command("git", "diff", "-U0", commit, "--", ".")
	cmd.Dir = gitRoot
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Git diff error: %v\n", err)
		return diffMap
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var currentFile string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "+++ b/") {
			rel := line[6:]
			currentFile = filepath.Join(gitRoot, rel)
			diffMap[currentFile] = FileChange{Lines: make(map[int]ChangeType)}
		} else if strings.HasPrefix(line, "@@ ") && currentFile != "" {
			re := regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
			match := re.FindStringSubmatch(line)
			if len(match) > 4 {
				oldLenStr := match[2]
				newStart, _ := strconv.Atoi(match[3])
				newLen, _ := strconv.Atoi(match[4])
				if match[4] == "" {
					newLen = 1
				}

				// Detect pure additions
				isNew := (oldLenStr == "0")
				changeType := New
				if !isNew {
					changeType = Modified
				}

				for i := 0; i < newLen; i++ {
					diffMap[currentFile].Lines[newStart+i] = changeType
				}
			}
		}
	}
	return diffMap
}

func processFile(path string, changes FileChange) string {
	content, _ := ioutil.ReadFile(path)
	lines := strings.Split(string(content), "\n")

	var result []string
	var currentPara []string
	paraChanged := Unchanged

	flushPara := func() {
		if len(currentPara) == 0 {
			return
		}
		paraText := strings.Join(currentPara, "\n")
		trimmed := strings.TrimSpace(paraText)

		isStructure := false
		for _, cmd := range []string{"\\chapter", "\\section", "\\subsection", "\\subsubsection", "\\item", "\\input", "\\include", "\\begin{table}", "\\end{table}", "\\begin{figure}", "\\end{figure}", "\\begin{equation}", "\\end{equation}", "\\begin{abstract}", "\\end{abstract}"} {
			if strings.HasPrefix(trimmed, cmd) {
				isStructure = true
				break
			}
		}

		if paraChanged != Unchanged && !isStructure && trimmed != "" && !strings.HasPrefix(trimmed, "%") {
			color := "Yellow"
			if paraChanged == New {
				color = "Green" // Standard XColor names
			}
			result = append(result, fmt.Sprintf("\n\\cbcolor{%s}\\cbstart\n%s\n\\cbend\n", color, paraText))
		} else {
			result = append(result, paraText)
		}
		currentPara = nil
		paraChanged = Unchanged
	}

	for i, line := range lines {
		lineNum := i + 1
		if strings.TrimSpace(line) == "" {
			flushPara()
			result = append(result, "")
			continue
		}

		if c, ok := changes.Lines[lineNum]; ok {
			if c == Modified || paraChanged == Unchanged {
				paraChanged = c
			}
		}
		currentPara = append(currentPara, line)
	}
	flushPara()

	return strings.Join(result, "\n")
}

func main() {
	commit := flag.String("commit", "HEAD^", "Git commit to compare against")
	output := flag.String("output", "diff_highlighted.pdf", "Output PDF name")
	flag.Parse()

	scriptDir, _ := os.Getwd()
	gitRootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	gitRootBytes, _ := gitRootCmd.Output()
	gitRoot := strings.TrimSpace(string(gitRootBytes))

	diffMap := getGitDiff(*commit, gitRoot)

	tmpDir := filepath.Join(scriptDir, "diff_tmp_go")
	os.RemoveAll(tmpDir)
	buildDir := filepath.Join(tmpDir, "build")
	os.MkdirAll(buildDir, 0755)

	filepath.Walk(scriptDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && (info.Name() == "diff_tmp_go" || info.Name() == "build") {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(scriptDir, path)
		target := filepath.Join(buildDir, rel)

		if info.IsDir() {
			os.MkdirAll(target, 0755)
			return nil
		}

		if filepath.Ext(path) == ".tex" {
			var newContent string
			if changes, ok := diffMap[path]; ok {
				newContent = processFile(path, changes)
			} else {
				c, _ := ioutil.ReadFile(path)
				newContent = string(c)
			}

			if rel == "practical-range-emptiness.tex" || rel == "proposal.tex" {
				// Robust preamble patch
				patch := `
\usepackage[dvipsnames,svgnames]{xcolor}
\usepackage[pdftex,color,leftbars]{changebar}
\setlength{\changebarwidth}{6pt}
\setlength{\changebarsep}{15pt}
`
				re := regexp.MustCompile(`(\\documentclass(?:\[[^\]]*\])?\{[^}]+\})`)
				newContent = re.ReplaceAllString(newContent, "$1"+patch)
			}
			ioutil.WriteFile(target, []byte(newContent), 0644)
		} else if info.Name() != "diff_highlighted.pdf" {
			input, _ := ioutil.ReadFile(path)
			ioutil.WriteFile(target, input, 0644)
		}
		return nil
	})

	fmt.Println("--- Compiling ---")
	mainFile := "practical-range-emptiness.tex"
	if _, err := os.Stat(filepath.Join(buildDir, mainFile)); os.IsNotExist(err) {
		mainFile = "proposal.tex"
	}

	// 3 runs to be safe
	for i := 1; i <= 3; i++ {
		cmd := exec.Command("latexmk", "-pdf", "-f", "-interaction=nonstopmode", mainFile)
		cmd.Dir = buildDir
		cmd.Run()
	}

	generatedPdf := filepath.Join(buildDir, strings.TrimSuffix(mainFile, ".tex")+".pdf")
	if _, err := os.Stat(generatedPdf); err == nil {
		data, _ := ioutil.ReadFile(generatedPdf)
		ioutil.WriteFile(filepath.Join(scriptDir, *output), data, 0644)
		fmt.Printf("\nSuccess! PDF saved to %s\n", *output)
	}
}
