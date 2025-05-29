# rexec

-`rexec` is a Go library that executes shell commands locally or over SSH, parses outputs into Go types, transfers files, and manages SSH connection retries and timeouts—without requiring agents on target machines.

By running ad-hoc commands and interpreting their results, you receive immediate feedback from remote hosts, when you can’t deploy or maintain persistent agents on remote hosts.

## Quick start
1. connection via SSH,
2. uploading the file via SFTP,
3. checking the existence of the file,
4. parsing the rights/owner/date of the file attribute via `ls`,
5. outputting the result from a Go structure.

```go
func main() {
	// 1. setting up ssh client
	sshCfg, err := ssh.NewConfig(
		"alice", "example.com", 22,
		ssh.WithPasswordAuth("secret"),
		ssh.WithRetry(3, 5*time.Second),
		ssh.WithKeepAlive(30*time.Second),
	)
	if err != nil {
		panic(err)
	}
	client, err := ssh.NewClient(sshCfg)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	// 2. upload file by SFTP
	data := []byte("Hello, rexec!")
	remoteDir := "/tmp/rexec"
	fileName := "hello.txt"
	spec := &rexec.FileSpec{
		TargetDir:  remoteDir,
		Filename:   fileName,
		Mode:       0644,
		FolderMode: 0755,
		Content:    &rexec.FileContent{Data: data},
	}
	// scp := ssh.NewSCPTransfer(client) // switch protocol is so simple
	sftp := ssh.NewSFTPTransfer(client)
	if err := sftp.Copy(ctx, spec); err != nil {
		panic(err)
	}
	
	// 3. check uploaded file existence
	var exists bool
	remotePath := path.Join(remoteDir, fileName)
	cmdExist := command.New(
		"test -f %s && echo true || echo false",
		command.WithArgs(remotePath),
		command.WithParser(&examples.PathExistence{}),
	)
	exists, err = rexec.RunParse[ssh.RunOption, bool](ctx, client, cmdExist)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Exists: %v\n", exists)
	
	// 4. gathering details of uploaded file
	var entries []examples.LsEntry
	cmdLs := command.New(
		"ls -la %s",
		command.WithArgs(remotePath),
		command.WithParser(&examples.LsParser{}),
	)
	entries, err = rexec.RunParse[ssh.RunOption, []examples.LsEntry](ctx, client, cmdLs)
	if err != nil {
		panic(err)
	}
	
	// 5. print result
	if len(entries) > 0 {
		e := entries[0]
		fmt.Printf("File: %s\n", e.Name)
		fmt.Printf("Owner: %s\n", e.Owner)
		fmt.Printf("Created: %s %s %s\n", e.Month, e.Day, e.TimeOrYear)
	}
}
```

## Features


- **Unified API**: Local and SSH execution via `Client[O any]` interface. Where `O` is `ssh.RunOption` or `local.RunOption`
- **Structured Parsing**: Convert command output into Go structs with `parser.Parser`
- **File Transfers**: Copy files using `FileSpec` over local FS, SCP, or SFTP
- **SSH Connection Retries**: Automatic dial retries on SSH connection failures
- **TCP Keep-Alive**: Prevent idle disconnections
- **Automatic PTY**: Allocate a pseudo-TTY for interactive commands (e.g. `sudo`, `passwd`)
- **Context-Aware**: Timeouts and cancellations via `context.Context`
- **Custom I/O Streams**: Override `stdin`, `stdout`, `stderr` for using in websockets, logs. Includes support for real-time streaming of output
- **Concurrency Safety**: Respects SSH server’s `MaxSessions` limit

---

## Installation

```bash
# Install the library
go get github.com/ngrsoftlab/rexec
```

## Configuration

### Local Client

```go
import "github.com/ngrsoftlab/rexec/local"

cfg := local.NewConfig().
  WithWorkDir("/tmp").       // default workdir
  WithEnvVars(map[string]string{ // environment for every run
    "GREETING": "Hello",
  })
client := local.NewClient(cfg)
defer client.Close()
```

- WithWorkDir(path string): set default workdir.
- WithEnvVars(map[string]string): set default environment.

### SSH Client

```go
import (
  "time"
  "github.com/ngrsoftlab/rexec/ssh"
)

sshCfg, err := ssh.NewConfig(
  "alice", "example.com", 22,
  ssh.WithPasswordAuth("secret"),           // password auth
  ssh.WithKnownHosts("~/.ssh/known_hosts"),
  ssh.WithRetry(3, 5*time.Second),          // SSH dial retry
  ssh.WithKeepAlive(30*time.Second),        // TCP keep-alive
  ssh.WithSudoPassword("sudoPass"),         // automatic sudo prompt
  ssh.WithWorkdir("/home/alice"),           // default remote dir
  ssh.WithMaxSessions(2),                   // concurrent sessions
)
if err != nil {
  // handle error
}
client, err := ssh.NewClient(sshCfg)
defer client.Close()
```

#### SSH Config Options
- `WithPort(int)`
- `WithTimeout(time.Duration)`
- `WithRetry(count int, interval time.Duration)` (SSH dial only)
- `WithKeepAlive(time.Duration)`
- `WithKnownHosts(path string)`
- `WithSudoPassword(string)`
- `WithEnvVars(map[string]string)`
- `WithWorkdir(string)`
- `WithMaxSessions(int)`
- Auth: 
    - `WithPasswordAuth(password string)`
    - `WithAgentAuth()`
    - `WithPrivateKeyPathAuth(path, passphrase string)`
    - `WithKeyBytesAuth([]byte, passphrase string)`


## Executing Commands

### Constructing Commands
```go
import "github.com/ngrsoftlab/rexec/command"

const listTpl = "ls -la %s"
cmd := command.New(
  listTpl,
  command.WithArgs("/var/log"),      // fmt.Sprintf args
  command.WithParser(&parser.LsParser{}), // parser
)
```

- `WithArgs(...any)`: append positional parameters.
- `WithParser(parser.Parser)`: attach parsing logic.

### Client.Run

```go
// dst is optional – pass nil to ignore parsing

// Example: local execution with override options
res, err := localClient.Run(ctx, cmd, &dst, local.WithWorkdir("/data"))
if err != nil {
// handle error; res.Stderr contains stderr
}
// Example: SSH execution with env var override
res, err = sshClient.Run(ctx, cmd, &dst, ssh.WithEnvVar("KEY", "value"))
if err != nil {
// handle error; res.ExitCode holds exit status
}
```
Local (local.RunOption):
- `WithWorkdir(string)`
- `WithEnvVar(key, value)`
- `WithStdout(io.Writer)`
- `WithStderr(io.Writer)`
- `WithStdin(io.Reader)`

SSH (ssh.RunOption):
- `WithEnvVar(key, value)`
- `WithStdout(io.Writer)`
- `WithStderr(io.Writer)`
- `WithStdin(io.Reader)`
- `WithStreaming()`: real-time output
- `WithoutBuffering()`: disable internal buffers


### Helpers & Generics

```go
import "github.com/ngrsoftlab/rexec"

// ignore parsing and return error only
err := rexec.RunNoResult[O](ctx, client, cmd, opts...)

// get raw outputs:
out, errOut, exit, err := rexec.RunRaw[O](ctx, client, cmd, opts...)

// parse into T:
dst, err := rexec.RunParse[O, T](ctx, client, cmd, opts...)
```

- O = local.RunOption or ssh.RunOption; 
- T = result type.

### Parsers
Implement parser.Parser to handle any command:

```go
type Parser interface {
  Parse(raw *RawResult, dst any) error
}
```
#### Built-in Parsers

- Located under parser/examples:
  - PathExistence: stdout "true"/"false" → bool
  - LsParser: parse ls -la → []LsEntry

#### Custom Parsers

```go
const uptimeTpl = "uptime -p" // create template

type UptimeInfo struct { Since string }

type UptimeParser struct{}

func (p *UptimeParser) Parse(raw *parser.RawResult, dst any) error {
  info, ok := dst.(*UptimeInfo)
  if !ok { return fmt.Errorf("dst must be *UptimeInfo") }
  info.Since = strings.TrimPrefix(raw.Stdout, "up ")
  return raw.Err
}

var info UptimeInfo
cmd := command.New(uptimeTpl, command.WithParser(&UptimeParser{}))
_, err := client.Run(ctx, cmd, &info)
```
## File Transfers

Use rexec.FileSpec:

```go
type FileSpec struct {
  TargetDir  string
  Filename   string
  Mode       os.FileMode
  FolderMode os.FileMode
  Content    *FileContent
}
```

### FileContent

`FileContent` supports three source types; choose one per `FileSpec`:

1. **In-Memory Data**  
   ```go
   content := &rexec.FileContent{Data: []byte("small payload")}
   ```

2. **Filesystem Path**  
   ```go
   content := &rexec.FileContent{SourcePath: "/path/to/file.txt"}
   ```

3. **Reader**  
   ```go
   f, _ := os.Open("/var/log/stream.log")
   content := &rexec.FileContent{Reader: f}
   ```
   • Use for large files or runtime-generated streams to avoid buffering overhead.

Internally, `ReaderAndSize()` returns:
```go
reader, size, err := content.ReaderAndSize()
```

The `FileContent.ReaderAndSize()` method encapsulates logic to produce an `io.ReadCloser` and its length. Its behavior depends on which field is set:

1. **`Data []byte`**
   - Returns `io.NopCloser(bytes.NewReader(Data))` and `int64(len(Data))`.
   - Zero-seeking overhead; length is known immediately.

2. **`SourcePath string`**
   - Opens the file via `os.Open(SourcePath)`.
   - Calls `File.Stat()` to get size, then returns file handle and size.
   - Errors if file does not exist or is inaccessible.

3. **`Reader io.Reader`**
   - If `Reader` implements `io.Seeker`, it seeks to determine current position and end to calculate 
   - If `Reader` is not seekable, returns error: "reader is not seekable".
   - Use this when you have a stream that supports seeking (e.g., `bytes.Reader`) or accept unknown size.

**Importance of Seekable Readers**
   - Seekable readers allow accurate `size` reporting, necessary for protocols like SCP that require upfront length.
   - Non-seekable streams must implement custom logic or be wrapped if size is needed

**Use Cases**
   - **In-memory**: when `Data` is small and performance matters.
   - **File path**: for large existing files; OS handles buffering.
   - **Seekable stream**: for random-access buffers or replayable streams.

### Local

```go
transfer := local.NewTransfer()
err := transfer.Copy(ctx, &rexec.FileSpec{...})
```

### SCP

```go
scp := ssh.NewSCPTransfer(sshClient)
err := scp.Copy(ctx, spec)
```

### SFTP

```go
sftp := ssh.NewSFTPTransfer(sshClient)
err := sftp.Copy(ctx, spec)
```

## SSH Connection Management & PTY

### Connection Options (SSH-only)
- `ssh.WithRetry(count int, interval time.Duration)`: retry SSH dialing up to count times with interval delay on connection failures; does not retry failed commands.
- `ssh.WithKeepAlive(duration time.Duration)`: send TCP keep-alive messages at the specified interval to keep the SSH connection alive.

### PTY Allocation & Sudo Handling
- `Automatic PTY`: commands containing keywords like `sudo`, `ssh`, or `docker login` trigger a pseudo-terminal allocation, enabling interactive prompts.
-` Sudo Password`: if `ssh.WithSudoPassword(password)` when set, the client monitors stdout for password: prompts and writes the provided password to stdin automatically.


## Session Limits

Ensures the SSH client never exceeds the host’s allowed concurrent sessions.
Recommended range 1-4 concurrent sessions. if problems occur, reduce the value
Configured via:
```go
    sshCfg, _ := ssh.NewConfig(
      "user", "host", 22,
      ssh.WithMaxSessions(3), // allow up to 3 simultaneous sessions
    )
```
- If the limit is reached, `OpenSession` blocks until a slot frees or the context expires.

## Stream Overrides and internal buffer control

Customize command I/O for live integrations and buffering control:

- `ssh.WithStreaming()`: immediately writes each chunk of stdout/stderr to your WithStdout/WithStderr writers as it arrives, rather than waiting for command completion. Use-cases:
- Live logs in a web dashboard or CLI progress indicators
- Pushing real-time output over WebSockets
- Interactive feedback loops in GUIs or monitoring tools 

Without streaming, output is accumulated internally and made available only after the command finishes.

- `ssh.WithoutBuffering()`: turns off the internal output buffers entirely; all data is sent directly to your writers. Benefits include:
- Reduced memory usage when handling large or continuous streams
- Predictable delivery in streaming pipelines or when chaining commands
      
If buffering is disabled and streaming is not enabled, the library will not store any output in `res.Stdout/res.Stderr` — all data goes to your writers.

Example of real-time WebSocket forwarding with minimal memory overhead:

```go
wsWriter := NewWebSocketWriter(conn)
res, err := sshClient.Run(
  ctx,
  cmd,
  nil,
  ssh.WithStdout(wsWriter),
  ssh.WithStderr(wsWriter),
  ssh.WithStreaming(),
  ssh.WithoutBuffering(),
)
if err != nil {
  // handle error; live output streamed via wsWriter
}
```

### Error Code Mapping

By default, `ExitCodeMapper` is applied automatically within every `Run` call, translating known exit codes into descriptive errors.  
For advanced use cases (e.g., custom error analysis), you can manually invoke the mapper:

```go
import "github.com/ngrsoftlab/rexec/utils"

// Run and receive raw result
res, err := sshClient.Run(ctx, cmd, nil)

// Default behavior: err includes mapped message
if err != nil {
  // err.Error() contains human-readable exit description
}

// Manual mapping example
mapper := utils.NewDefaultExitCodeMapper()
if res.ExitCode != 0 {
  desc := mapper.Lookup(res.ExitCode)
  log.Printf("Custom error mapping: code %d => %s", res.ExitCode, desc)
}
```

Use manual mapping when you need to log, categorize, or transform exit statuses beyond the default error message.



© 2025 NGRSOFTLAB
