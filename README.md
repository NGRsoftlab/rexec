# rexec

A lightweight Go framework for **local** (with SSH support coming soon) command execution featuring:

- **Templated commands** using `fmt.Sprintf`-style placeholders
- **Context-aware execution** with built-in support for timeouts and cancellation
- **Per-run overrides** for working directory and environment variables
- **Structured results** via the `RawResult` type
- **Extensible parsers** to convert raw output into Go types

---

## Installation

```bash
go get github.com/ngrsoftlab/rexec
```

## 1. Configuration

```go
import "github.com/ngrsoftlab/rexec/config"

// Create a local config with a default working directory and environment variables
cfg := config.NewLocalConfig().
WithWorkDir("/tmp").
WithEnvVars(map[string]string{
"GREETING": "Hello",
"TARGET":   "Gopher",
})
```

- **WorkDir**: the default working directory for all commands
- **EnvVars**: additional environment variables for commands

## 2. Creating a Session

```go
import "github.com/ngrsoftlab/rexec"

// Create a new local session and ensure it gets closed
sess := rexec.NewLocalSession(cfg)
defer sess.Close()
```

## 3. Running Commands

### 3.1 Simple Command

```go
import "github.com/ngrsoftlab/rexec/command"

cmd := command.New("touch %s && echo $(pwd)", "demo.txt")
res, err := sess.Run(context.Background(), cmd, nil)
if err != nil {
    log.Fatalf("Error: %v, stderr: %s", err, res.Stderr)
}
fmt.Println("Created file in:", res.Stdout)
```

### 3.2 Parsing Output

```go
import "github.com/ngrsoftlab/rexec/parser"

cmd := command.NewWithParser(
`[ -f %s ] && echo true || echo false`,
&parser.PathExistence{},
"demo.txt",
)

var exists bool
res, err := sess.Run(context.Background(), cmd, &exists)
if err != nil {
    log.Fatalf("Error: %v, stderr: %s", err, res.Stderr)
}

fmt.Printf("File exists: %t (exit code %d)n", exists, res.ExitCode)
```

### 3.3 One-time Overrides

```go
cmd := command.New(`echo "$GREETING, $TARGET! Dir: $(pwd)"`)
res, err := sess.Run(context.Background(), cmd, nil, rexec.WithWorkdir("/home"), rexec.WithEnvVar("TARGET","Analytics"))
if err != nil {
    log.Fatalf("Error: %v, stderr: %s", err, res.Stderr)
}

fmt.Print(res.Stdout)
```

## 4. Timeouts and Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

cmd := command.New("sleep 2")
res, err := sess.Run(ctx, cmd, nil)
if err != nil {
    fmt.Printf("Timed out as expected: %v (exit code %d)n", err, res.ExitCode)
}

```

## 5. `RawResult` Fields

| Field      | Description                                                  |
|------------|--------------------------------------------------------------|
| `Command`  | The fully rendered command string                            |
| `Stdout`   | Captured standard output                                     |
| `Stderr`   | Captured standard error                                      |
| `ExitCode` | Process exit code (`0` = success, `-1` = canceled)         |
| `Duration` | Execution time                                              |
| `Err`      | Execution, timeout, or parsing error                        |

---

© 2025 NGRSOFTLAB
