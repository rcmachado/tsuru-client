// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/fs"
	"launchpad.net/gnuflag"
)

const keyTruncate = 60

type keyReader struct {
	fsystem fs.Fs
}

func (r *keyReader) fs() fs.Fs {
	if r.fsystem == nil {
		r.fsystem = fs.OsFs{}
	}
	return r.fsystem
}

func (r *keyReader) readKey(context *cmd.Context) (string, error) {
	keyPath := context.Args[1]
	var input io.Reader
	if keyPath == "-" {
		input = context.Stdin
	} else {
		f, err := r.fs().Open(keyPath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		input = f
	}
	output, err := ioutil.ReadAll(input)
	return string(output), err
}

type keyAdd struct {
	keyReader
}

func (c *keyAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "key-add",
		Usage:   "key-add <key-name> <path/to/key/file.pub>",
		Desc:    "adds a public key to your account",
		MinArgs: 2,
	}
}

func (c *keyAdd) Run(context *cmd.Context, client *cmd.Client) error {
	keyName := context.Args[0]
	keyPath := context.Args[1]
	key, err := c.readKey(context)
	if os.IsNotExist(err) {
		return fmt.Errorf("file %q doesn't exist", keyPath)
	} else if err != nil {
		return err
	}
	jsonBody := fmt.Sprintf(`{"key":%q,"name":%q}`, strings.Replace(key, "\n", "", -1), keyName)
	url, err := cmd.GetURL("/users/keys")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, bytes.NewBufferString(jsonBody))
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Key %q successfully added!\n", keyName)
	return nil
}

type keyRemove struct{}

func (c *keyRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "key-remove",
		Usage:   "key-remove <key-name>",
		Desc:    "removes the given public key from your account",
		MinArgs: 1,
	}
}

func (c *keyRemove) Run(context *cmd.Context, client *cmd.Client) error {
	b := bytes.NewBufferString(fmt.Sprintf(`{"name":%q}`, context.Args[0]))
	url, err := cmd.GetURL("/users/keys")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, b)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Key %q successfully removed!\n", context.Args[0])
	return nil
}

type keyList struct {
	notrunc bool
	fs      *gnuflag.FlagSet
}

func (c *keyList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "key-list",
		Usage: "key-list [-n/--no-truncate]",
		Desc:  "lists public keys registered in your account",
	}
}

func (c *keyList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/keys")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var keys map[string]string
	err = json.NewDecoder(resp.Body).Decode(&keys)
	if err != nil {
		return err
	}
	var table cmd.Table
	table.Headers = cmd.Row{"Name", "Content"}
	table.LineSeparator = c.notrunc
	for name, content := range keys {
		row := []string{name, content}
		if !c.notrunc && len(row[1]) > keyTruncate {
			row[1] = row[1][:keyTruncate] + "..."
		}
		table.AddRow(cmd.Row(row))
	}
	table.SortByColumn(0)
	context.Stdout.Write(table.Bytes())
	return nil
}

func (c *keyList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("key-list", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.notrunc, "n", false, "disable truncation of key content")
		c.fs.BoolVar(&c.notrunc, "no-truncate", false, "disable truncation of key content")
	}
	return c.fs
}
