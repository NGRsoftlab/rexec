// Copyright © NGRSoftlab 2020-2025

package main

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/ngrsoftlab/rexec"
	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
	"github.com/ngrsoftlab/rexec/parser/examples"
	"github.com/ngrsoftlab/rexec/ssh"
)

func main() {
	// 1. setting ip ssh client
	sshCfg, err := ssh.NewConfig(
		"alice", "example.ip", 22, // to test - change credits
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
		command.WithParser(&examples.BoolParser{}),
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

	// PARSING RESULTS IN VARS FORM BATCH EXECUTION

	cmdList := []*command.Command{
		cmdExist,
		cmdLs,
	}

	results := make([]*parser.RawResult, 0, len(cmdList))
	for _, cmd := range cmdList {
		res, err := client.Run(ctx, cmd, nil)
		if err != nil {
			panic(err)
		}
		results = append(results, res)
	}

	var boolVar bool
	var lsEntries []examples.LsEntry

	mappingVars := map[*command.Command]any{
		cmdLs:    &lsEntries,
		cmdExist: &boolVar,
	}

	if err := rexec.ApplyParsers(results, mappingVars); err != nil {
		panic(err)
	}

	// OR YOU CAN MANUALLY CREATE COMMAND->RAWRESULT MAPPING

	rawMap := make(map[*command.Command]*parser.RawResult, len(results))
	for i, cmd := range cmdList {
		rawMap[cmd] = results[i]
	}

	if err := rexec.ParseWithMapping(rawMap, mappingVars); err != nil {
		panic(err)
	}

	fmt.Printf("Exists: %v\n", boolVar)
	if len(lsEntries) > 0 {
		e := lsEntries[0]
		fmt.Printf("File: %s, Owner: %s, Created: %s %s %s\n",
			e.Name, e.Owner, e.Month, e.Day, e.TimeOrYear)
	}
}
