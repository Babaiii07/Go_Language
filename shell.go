package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"golang.org/x/term"
)

var builtinCMDs = []string{
	"exit",
	"echo",
	"type",
	"pwd",
	"cd",
	"clear",
	"ls",
	"cat",
	"cp",
	"mv",
	"rm",
	"mkdir",
	"rmdir",
}

type CMD struct {
	Name   string
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
}

func main() {
	for {
		printPrompt()
		input := readInputWithAutocomplete(os.Stdin)
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		fields := parseCommand(input)
		if len(fields) == 0 {
			continue
		}

		fields, stdoutFile, stderrFile, stdoutAppend, stderrAppend := processRedirectionOperators(fields)
		if len(fields) == 0 {
			continue
		}

		builtins := map[string]func([]string){
			"exit":  exitHandler,
			"echo":  echoHandler,
			"type":  typeHandler,
			"pwd":   pwdHandler,
			"cd":    cdHandler,
			"clear": clearHandler,
			"ls":    lsHandler,
			"cat":   catHandler,
			"cp":    cpHandler,
			"mv":    mvHandler,
			"rm":    rmHandler,
			"mkdir": mkdirHandler,
			"rmdir": rmdirHandler,
		}

		if handler, exists := builtins[fields[0]]; exists {
			executeBuiltinWithRedirection(handler, fields, stdoutFile, stderrFile, stdoutAppend, stderrAppend)
		} else {
			executeExternalWithRedirection(fields, stdoutFile, stderrFile, stdoutAppend, stderrAppend)
		}
	}
}

func printPrompt() {
	fmt.Fprint(os.Stdout, "\r\033[1;32mShX\033[0m ➜ ")
}

var lastTabPrefix string
var tabPressCount int

func readInputWithAutocomplete(rd *os.File) string {
	oldState, err := term.MakeRaw(int(rd.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(rd.Fd()), oldState)

	var input string
	r := bufio.NewReader(rd)
	for {
		rn, _, err := r.ReadRune()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			continue
		}

		switch rn {
		case '\x03':
			term.Restore(int(rd.Fd()), oldState)
			os.Exit(0)
		case '\r', '\n':
			fmt.Fprint(os.Stdout, "\r\n")

			tabPressCount = 0
			lastTabPrefix = ""
			return input
		case '\x7F':
			if len(input) > 0 {
				input = input[:len(input)-1]
			}
			tabPressCount = 0
			lastTabPrefix = ""
			printPromptWithInput(input)
		case '\t':
			prefix := input
			if strings.Contains(prefix, " ") {
				tabPressCount = 0
				lastTabPrefix = ""
				break
			}

			if prefix != lastTabPrefix {
				tabPressCount = 0
				lastTabPrefix = prefix
			}

			tabPressCount++
			result, matches := autocomplete(prefix, tabPressCount)

			if len(matches) > 1 && tabPressCount > 1 && result == "" {
				printAllMatches(matches)
				printPromptWithInput(input)
			} else if result != "" {
				input += result
				if len(matches) == 1 {
					input += " "
				}
				printPromptWithInput(input)
				tabPressCount = 0
				lastTabPrefix = ""
			} else {
				fmt.Fprint(os.Stdout, "\a")
			}
		default:
			input += string(rn)
			tabPressCount = 0
			lastTabPrefix = ""
			printPromptWithInput(input)
		}
	}
}

func printAllMatches(matches []string) {
	fmt.Fprint(os.Stdout, "\r\n")
	fmt.Fprint(os.Stdout, strings.Join(matches, "  "))
	fmt.Fprint(os.Stdout, "\r\n")
}

func printPromptWithInput(input string) {
	fmt.Fprint(os.Stdout, "\r\x1b[K\033[1;32mShX\033[0m ➜ "+input)
}

func autocomplete(prefix string, tabCount int) (string, []string) {
	if prefix == "" {
		return "", nil
	}

	var matches []string

	for _, cmd := range builtinCMDs {
		if strings.HasPrefix(cmd, prefix) && cmd != prefix {
			matches = append(matches, cmd)
		}
	}

	if len(matches) == 0 {
		pathEnv := os.Getenv("PATH")
		dirs := append([]string{"."}, strings.Split(pathEnv, ":")...)
		found := make(map[string]bool)
		for _, dir := range dirs {
			files, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, file := range files {

				if file.IsDir() {
					continue
				}
				name := file.Name()

				if strings.HasPrefix(name, prefix) && name != prefix {
					if !found[name] {
						found[name] = true
						matches = append(matches, name)
					}
				}
			}
		}
	}

	if len(matches) == 0 {
		return "", nil
	}

	sort.Strings(matches)

	if len(matches) == 1 {
		return strings.TrimPrefix(matches[0], prefix), matches
	}

	lcp := longestCommonPrefix(matches)
	if len(lcp) > len(prefix) {
		return lcp[len(prefix):], matches
	}

	if tabCount > 1 {
		return "", matches
	}

	return "", nil
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			if prefix == "" {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func processRedirectionOperators(fields []string) ([]string, string, string, bool, bool) {
	stdoutFile := ""
	stderrFile := ""
	stdoutAppend := false
	stderrAppend := false
	var finalFields []string

	for i := 0; i < len(fields); {
		token := fields[i]
		switch token {
		case ">>", "1>>":
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stdoutFile = fields[i+1]
			stdoutAppend = true
			i += 2
		case ">", "1>":
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stdoutFile = fields[i+1]
			i += 2
		case "2>":
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stderrFile = fields[i+1]
			i += 2
		case "2>>":
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stderrFile = fields[i+1]
			stderrAppend = true
			i += 2
		default:
			finalFields = append(finalFields, token)
			i++
		}
	}
	return finalFields, stdoutFile, stderrFile, stdoutAppend, stderrAppend
}

func executeBuiltinWithRedirection(
	handler func([]string),
	args []string,
	stdoutFile, stderrFile string,
	stdoutAppend, stderrAppend bool,
) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	if stdoutFile != "" {
		file, err := openFile(stdoutFile, stdoutAppend)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stdout redirection:", err)
			return
		}
		os.Stdout = file
		defer func() {
			os.Stdout = oldStdout
			file.Close()
		}()
	}

	if stderrFile != "" {
		file, err := openFile(stderrFile, stderrAppend)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", err)
			return
		}
		os.Stderr = file
		defer func() {
			os.Stderr = oldStderr
			file.Close()
		}()
	}

	handler(args)
}

func executeExternalWithRedirection(
	fields []string,
	stdoutFile, stderrFile string,
	stdoutAppend, stderrAppend bool,
) {
	if stdoutFile == "" && stderrFile == "" {
		executeCommand(fields)
		return
	}

	path, err := exec.LookPath(fields[0])
	if err != nil {
		outputError(fields[0], stderrFile, stderrAppend)
		return
	}

	cmd := exec.Command(path, fields[1:]...)

	if stdoutFile != "" {
		if file, err := openFile(stdoutFile, stdoutAppend); err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stdout redirection:", err)
			return
		} else {
			cmd.Stdout = file
			defer file.Close()
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if stderrFile != "" {
		if file, err := openFile(stderrFile, stderrAppend); err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", err)
			return
		} else {
			cmd.Stderr = file
			defer file.Close()
		}
	} else {
		cmd.Stderr = os.Stderr
	}

	cmd.Run()
}

func executeCommand(fields []string) {
	cmd := exec.Command(fields[0], fields[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(fields[0] + ": command not found")
	}
}

func openFile(fileName string, appendMode bool) (*os.File, error) {
	if appendMode {
		return os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	return os.Create(fileName)
}

func outputError(cmdName, stderrFile string, appendMode bool) {
	message := cmdName + ": command not found"
	if stderrFile != "" {
		if file, err := openFile(stderrFile, appendMode); err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", err)
		} else {
			fmt.Fprintln(file, message)
			file.Close()
		}
	} else {
		fmt.Fprintln(os.Stderr, message)
	}
}

func parseCommand(command string) []string {
	var result []string
	var current strings.Builder
	inSingleQuote, inDoubleQuote, escaped := false, false, false

	for i := 0; i < len(command); i++ {
		c := command[i]

		if escaped {
			if inDoubleQuote && c != '$' && c != '`' && c != '"' && c != '\\' && c != '\n' {
				current.WriteByte('\\')
			}
			current.WriteByte(c)
			escaped = false
			continue
		}

		switch c {
		case '\\':
			if inSingleQuote {
				current.WriteByte(c)
			} else {
				escaped = true
			}
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			} else {
				current.WriteByte(c)
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			} else {
				current.WriteByte(c)
			}
		case ' ':
			if !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(c)
			}
		default:
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

func exitHandler(args []string) {
	os.Exit(0)
}

func echoHandler(args []string) {
	fmt.Println(strings.Join(args[1:], " "))
}

func typeHandler(args []string) {
	if len(args) < 2 {
		fmt.Println("type: missing argument")
		return
	}
	cmd := args[1]
	builtins := map[string]bool{
		"echo": true,
		"exit": true,
		"type": true,
		"pwd":  true,
		"cd":   true,
	}

	if builtins[cmd] {
		fmt.Println(cmd + " is a shell builtin")
	} else if path, err := exec.LookPath(cmd); err == nil {
		fmt.Println(cmd + " is " + path)
	} else {
		fmt.Println(cmd + ": not found")
	}
}

func pwdHandler(args []string) {
	cwd, _ := os.Getwd()
	fmt.Println(cwd)
}

func cdHandler(args []string) {
	if len(args) < 2 {
		fmt.Println("cd: missing argument")
		return
	}

	dir := args[1]
	switch {
	case dir == "~":
		dir = os.Getenv("HOME")
	case strings.HasPrefix(dir, "~/"):
		dir = os.Getenv("HOME") + dir[1:]
	}

	if err := os.Chdir(dir); err != nil {
		fmt.Printf("cd: %s: No such file or directory\n", dir)
	}
}

func clearHandler(args []string) {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func lsHandler(args []string) {
	dir := "."
	if len(args) > 1 {
		dir = args[1]
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("ls: cannot access '%s': %v\n", dir, err)
		return
	}

	for _, file := range files {
		fmt.Println(file.Name())
	}
}

func catHandler(args []string) {
	if len(args) < 2 {
		fmt.Println("cat: missing file operand")
		return
	}

	for _, file := range args[1:] {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("cat: %s: %v\n", file, err)
			continue
		}
		fmt.Print(string(content))
	}
}

func cpHandler(args []string) {
	if len(args) < 3 {
		fmt.Println("cp: missing file operand")
		return
	}

	src := args[1]
	dst := args[2]

	srcFile, err := os.ReadFile(src)
	if err != nil {
		fmt.Printf("cp: cannot stat '%s': %v\n", src, err)
		return
	}

	err = os.WriteFile(dst, srcFile, 0644)
	if err != nil {
		fmt.Printf("cp: cannot create '%s': %v\n", dst, err)
		return
	}
}

func mvHandler(args []string) {
	if len(args) < 3 {
		fmt.Println("mv: missing file operand")
		return
	}

	src := args[1]
	dst := args[2]

	err := os.Rename(src, dst)
	if err != nil {
		fmt.Printf("mv: cannot move '%s' to '%s': %v\n", src, dst, err)
		return
	}
}

func rmHandler(args []string) {
	if len(args) < 2 {
		fmt.Println("rm: missing operand")
		return
	}

	for _, file := range args[1:] {
		err := os.Remove(file)
		if err != nil {
			fmt.Printf("rm: cannot remove '%s': %v\n", file, err)
		}
	}
}

func mkdirHandler(args []string) {
	if len(args) < 2 {
		fmt.Println("mkdir: missing operand")
		return
	}

	for _, dir := range args[1:] {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Printf("mkdir: cannot create directory '%s': %v\n", dir, err)
		}
	}
}

func rmdirHandler(args []string) {
	if len(args) < 2 {
		fmt.Println("rmdir: missing operand")
		return
	}

	for _, dir := range args[1:] {
		err := os.Remove(dir)
		if err != nil {
			fmt.Printf("rmdir: failed to remove '%s': %v\n", dir, err)
		}
	}
}
