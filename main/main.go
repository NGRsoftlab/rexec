// Copyright © NGRSoftlab 2020-2025

package main

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/ngrsoftlab/rexec"
	"github.com/ngrsoftlab/rexec/command"
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
