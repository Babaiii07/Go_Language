# ShX Shell

A lightweight shell implementation in Go that provides a Unix-like command-line interface with built-in commands and external command execution capabilities.

## Features

### Built-in Commands
- `exit`: Exit the shell
- `echo`: Display text
- `type`: Show command type (builtin or external)
- `pwd`: Print working directory
- `cd`: Change directory
- `clear`: Clear the terminal screen
- `ls`: List directory contents
- `cat`: Display file contents
- `cp`: Copy files
- `mv`: Move/rename files
- `rm`: Remove files
- `mkdir`: Create directories
- `rmdir`: Remove empty directories

### Advanced Features
- Command autocompletion (press Tab)
- Input redirection support:
  - `>` for stdout redirection
  - `>>` for stdout redirection with append
  - `2>` for stderr redirection
  - `2>>` for stderr redirection with append
- Command history
- Path-based command execution
- Error handling and reporting

## Usage

### Basic Commands
```bash
# List directory contents
ls

# Change directory
cd /path/to/directory

# Display current directory
pwd

# Display file contents
cat file.txt

# Create directory
mkdir newdir

# Remove file
rm file.txt
```

### Redirection Examples
```bash
# Redirect output to file
ls > output.txt

# Append output to file
echo "new line" >> output.txt

# Redirect error to file
command 2> error.log

# Redirect both output and error
command > output.txt 2>&1
```

### Autocompletion
- Press Tab to autocomplete commands and filenames
- Press Tab twice to show all possible completions

## Implementation Details

The shell is implemented in Go and uses the following key components:

1. **Command Parsing**
   - Handles quoted strings (single and double quotes)
   - Supports escape characters
   - Processes redirection operators

2. **Command Execution**
   - Built-in command handling
   - External command execution via PATH
   - Redirection support for both stdout and stderr

3. **User Interface**
   - Custom prompt with color support
   - Tab completion
   - Command history

## Dependencies

- `golang.org/x/term`: For terminal handling and raw mode input

## Building

```bash
go build shell.go
```

## Running

```bash
./shell
```

## Error Handling

The shell provides clear error messages for common scenarios:
- Command not found
- File/directory not found
- Permission denied
- Invalid syntax

## Notes

- The shell is designed to work on Windows systems
- Some Unix-specific commands may have slightly different behavior
- The shell uses the system's PATH environment variable for external command execution

## License

This project is open source and available under the MIT License.