package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/ngrsoftlab/rexec"
	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/config"
	"github.com/ngrsoftlab/rexec/parser/examples"
)

func main() {

	localConfig := config.NewLocalConfig().
		WithWorkDir("/tmp").
		WithEnvVars(map[string]string{
			"GREETING": "Hi",
			"TARGET":   "Developer",
		})

	sess := rexec.NewLocalSession(localConfig)

	touchCommand := command.New("touch %s && echo $(pwd)", command.WithArgs("demo.txt"))
	touchResult, err := sess.Run(context.Background(), touchCommand, nil)
	if err != nil {
		fmt.Printf("create file failed: %v\nstderr: %s\n", err, touchResult.Stderr)
	} else {
		fmt.Printf("Created demo.txt\nWorkdir is: %s", touchResult.Stdout)
	}

	checkFileExistsCommand := command.New(`[ -f %s ] && echo true || echo false`,
		command.WithParser(&examples.PathExistence{}),
		command.WithArgs("demo.txt"),
	)

	var exists bool
	checkFileResult, err := sess.Run(context.Background(), checkFileExistsCommand, &exists)
	if err != nil {
		fmt.Printf("exists check failed: %v\nstderr: %s\n", err, checkFileResult.Stderr)
	} else {
		fmt.Printf("demo.txt exists: %t (exit code %d)\n", exists, checkFileResult.ExitCode)
	}

	printEnvCommand := command.New(`echo "$GREETING, $TARGET!" Working dir: $(pwd)`)
	printEnvResult, err := sess.Run(
		context.Background(),
		printEnvCommand,
		nil,
		rexec.WithWorkdir("/home"),
		rexec.WithEnvVar("TARGET", "Analytics"),
	)
	if err != nil {
		fmt.Printf("print failed: %v\nstderr: %s\n", err, printEnvResult.Stderr)
	} else {
		fmt.Printf("%s", printEnvResult.Stdout)
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	sleepCommand := command.New("sleep 2")
	if sleepResult, err := sess.Run(timeoutCtx, sleepCommand, nil); err != nil {
		fmt.Printf("timeout as expected: %v (exit %d)\n", err, sleepResult.ExitCode)
	} else {
		fmt.Printf("sleep finished (unexpected): %s\n", sleepResult.Stdout)
	}

	f, _ := os.Create("out.log")
	defer f.Close()

	lsCommand := command.New("ls -la %s",
		command.WithParser(&examples.LsParser{}),
		command.WithArgs("/home/asysoyev"),
	)

	var entries []examples.LsEntry
	lsResult, err := sess.Run(context.Background(), lsCommand, &entries)
	if err != nil {
		fmt.Printf("ls failed: %v\nstderr: %s\n", err, lsResult.Stderr)
		return
	}

	stats := make(map[string]struct {
		Count     int
		TotalSize int64
	})

	for _, entry := range entries {
		mode, err := entry.ParsePermissions()
		if err != nil {
			fmt.Printf("failed to parse permissions: %v\n", err)
		}

		if !mode.IsRegular() {
			continue
		}

		s := stats[entry.Owner]
		s.Count++
		s.TotalSize += entry.Size
		stats[entry.Owner] = s
	}

	owners := make([]string, 0, len(stats))
	for owner := range stats {
		owners = append(owners, owner)
	}
	sort.Strings(owners)

	fmt.Printf("%-10s %-11s %s\n", "OWNER", "FILES_COUNT", "TOTAL_SIZE")
	for _, owner := range owners {
		s := stats[owner]
		fmt.Printf("%-10s %-11d %d\n", owner, s.Count, s.TotalSize)
	}
}
