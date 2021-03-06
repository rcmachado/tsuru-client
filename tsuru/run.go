// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"launchpad.net/gnuflag"
)

type appRun struct {
	cmd.GuessingCommand
	fs   *gnuflag.FlagSet
	once bool
}

func (c *appRun) Info() *cmd.Info {
	desc := `run a command in all instances of the app, and prints the output.

If you use the '--once' flag tsuru will run the command only in one unit.

If you don't provide the app name, tsuru will try to guess it.
`
	return &cmd.Info{
		Name:    "app-run",
		Usage:   "app-run <command> [commandarg1] [commandarg2] ... [commandargn] [-a/--app appname] [-o/--once]",
		Desc:    desc,
		MinArgs: 1,
	}
}

func (c *appRun) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/run?once=%t", appName, c.once))
	if err != nil {
		return err
	}
	b := strings.NewReader(strings.Join(context.Args, " "))
	request, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	r, err := client.Do(request)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, r.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

func (c *appRun) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.BoolVar(&c.once, "once", false, "Running only one unit")
		c.fs.BoolVar(&c.once, "o", false, "Running only one unit")
	}
	return c.fs
}
