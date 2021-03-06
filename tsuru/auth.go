// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

	"github.com/tsuru/tsuru/cmd"
	"golang.org/x/crypto/ssh/terminal"
	"launchpad.net/gnuflag"
)

type userCreate struct{}

func (c *userCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "user-create",
		Usage:   "user-create <email>",
		Desc:    "creates a user.",
		MinArgs: 1,
	}
}

func (c *userCreate) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users")
	if err != nil {
		return err
	}
	email := context.Args[0]
	fmt.Fprint(context.Stdout, "Password: ")
	password, err := passwordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "\nConfirm: ")
	confirm, err := passwordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout)
	if password != confirm {
		return errors.New("Passwords didn't match.")
	}
	b := bytes.NewBufferString(`{"email":"` + email + `", "password":"` + password + `"}`)
	request, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if resp != nil {
		if resp.StatusCode == http.StatusNotFound ||
			resp.StatusCode == http.StatusMethodNotAllowed {
			return errors.New("User creation is disabled.")
		}
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `User "%s" successfully created!`+"\n", email)
	return nil
}

type userRemove struct{}

func (c *userRemove) Run(context *cmd.Context, client *cmd.Client) error {
	var answer string
	fmt.Fprint(context.Stdout, `Are you sure you want to remove your user from tsuru? (y/n) `)
	fmt.Fscanf(context.Stdin, "%s", &answer)
	if answer != "y" {
		fmt.Fprintln(context.Stdout, "Abort.")
		return nil
	}
	url, err := cmd.GetURL("/users")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	filesystem().Remove(cmd.JoinWithUserDir(".tsuru_token"))
	fmt.Fprint(context.Stdout, "User successfully removed.\n")
	return nil
}

func (c *userRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "user-remove",
		Usage:   "user-remove",
		Desc:    "removes your user from tsuru server.",
		MinArgs: 0,
	}
}

type teamCreate struct{}

func (c *teamCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-create",
		Usage:   "team-create <teamname>",
		Desc:    "creates a new team.",
		MinArgs: 1,
	}
}

func (c *teamCreate) Run(context *cmd.Context, client *cmd.Client) error {
	team := context.Args[0]
	b := bytes.NewBufferString(fmt.Sprintf(`{"name":"%s"}`, team))
	url, err := cmd.GetURL("/teams")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `Team "%s" successfully created!`+"\n", team)
	return nil
}

type teamRemove struct {
	cmd.ConfirmationCommand
}

func (c *teamRemove) Run(context *cmd.Context, client *cmd.Client) error {
	team := context.Args[0]
	question := fmt.Sprintf("Are you sure you want to remove team %q?", team)
	if !c.Confirm(context, question) {
		return nil
	}
	url, err := cmd.GetURL(fmt.Sprintf("/teams/%s", team))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `Team "%s" successfully removed!`+"\n", team)
	return nil
}

func (c *teamRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-remove",
		Usage:   "team-remove <team-name>",
		Desc:    "removes a team from tsuru server.",
		MinArgs: 1,
	}
}

type teamUserAdd struct{}

func (c *teamUserAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-user-add",
		Usage:   "team-user-add <teamname> <useremail>",
		Desc:    "adds a user to a team.",
		MinArgs: 2,
	}
}

func (c *teamUserAdd) Run(context *cmd.Context, client *cmd.Client) error {
	teamName, userName := context.Args[0], context.Args[1]
	url, err := cmd.GetURL(fmt.Sprintf("/teams/%s/%s", teamName, userName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `User "%s" was added to the "%s" team`+"\n", userName, teamName)
	return nil
}

type teamUserRemove struct{}

func (c *teamUserRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-user-remove",
		Usage:   "team-user-remove <teamname> <useremail>",
		Desc:    "removes a user from a team.",
		MinArgs: 2,
	}
}

func (c *teamUserRemove) Run(context *cmd.Context, client *cmd.Client) error {
	teamName, userName := context.Args[0], context.Args[1]
	url, err := cmd.GetURL(fmt.Sprintf("/teams/%s/%s", teamName, userName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `User "%s" was removed from the "%s" team`+"\n", userName, teamName)
	return nil
}

type teamUserList struct{}

func (teamUserList) Run(context *cmd.Context, client *cmd.Client) error {
	teamName := context.Args[0]
	url, err := cmd.GetURL("/teams/" + teamName)
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
	var t struct{ Users []string }
	err = json.NewDecoder(resp.Body).Decode(&t)
	if err != nil {
		return err
	}
	sort.Strings(t.Users)
	for _, user := range t.Users {
		fmt.Fprintf(context.Stdout, "- %s\n", user)
	}
	return nil
}

func (teamUserList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-user-list",
		Usage:   "team-user-list <teamname>",
		Desc:    "List members of a team.",
		MinArgs: 1,
	}
}

type teamList struct{}

func (c *teamList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-list",
		Usage:   "team-list",
		Desc:    "List all teams that you are member.",
		MinArgs: 0,
	}
}

func (c *teamList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/teams")
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
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var teams []map[string]string
		err = json.Unmarshal(b, &teams)
		if err != nil {
			return err
		}
		io.WriteString(context.Stdout, "Teams:\n\n")
		for _, team := range teams {
			fmt.Fprintf(context.Stdout, "  - %s\n", team["name"])
		}
	}
	return nil
}

type changePassword struct{}

func (c *changePassword) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/password")
	if err != nil {
		return err
	}
	var body bytes.Buffer
	fmt.Fprint(context.Stdout, "Current password: ")
	old, err := passwordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "\nNew password: ")
	new, err := passwordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "\nConfirm: ")
	confirm, err := passwordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout)
	if new != confirm {
		return errors.New("New password and password confirmation didn't match.")
	}
	jsonBody := map[string]string{
		"old": old,
		"new": new,
	}
	err = json.NewEncoder(&body).Encode(jsonBody)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, &body)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Password successfully updated!")
	return nil
}

func (c *changePassword) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "change-password",
		Usage: "change-password",
		Desc:  "Change your password.",
	}
}

type resetPassword struct {
	token string
}

func (c *resetPassword) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "reset-password",
		Usage: "reset-password <email> [--token|-t <token>]",
		Desc: `Resets the user password.

This process is composed of two steps:

1. Generate a new token
2. Reset the password using the token

In order to generate the token, users should run this command without the --token flag.
The token will be mailed to the user.

With the token in hand, the user can finally reset the password using the --token flag.
The new password will also be mailed to the user.`,
		MinArgs: 1,
	}
}

func (c *resetPassword) msg() string {
	if c.token == "" {
		return `You've successfully started the password reset process.

Please check your email.`
	}
	return `Your password has been reset and mailed to you.

Please check your email.`
}

func (c *resetPassword) Run(context *cmd.Context, client *cmd.Client) error {
	url := fmt.Sprintf("/users/%s/password", context.Args[0])
	if c.token != "" {
		url += "?token=" + c.token
	}
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, _ := http.NewRequest("POST", url, nil)
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, c.msg())
	return nil
}

func (c *resetPassword) Flags() *gnuflag.FlagSet {
	fs := gnuflag.NewFlagSet("reset-password", gnuflag.ExitOnError)
	fs.StringVar(&c.token, "token", "", "Token to reset the password")
	fs.StringVar(&c.token, "t", "", "Token to reset the password")
	return fs
}

func passwordFromReader(reader io.Reader) (string, error) {
	var (
		password []byte
		err      error
	)
	if file, ok := reader.(*os.File); ok && terminal.IsTerminal(int(file.Fd())) {
		password, err = terminal.ReadPassword(int(file.Fd()))
		if err != nil {
			return "", err
		}
	} else {
		fmt.Fscanf(reader, "%s\n", &password)
	}
	if len(password) == 0 {
		msg := "You must provide the password!"
		return "", errors.New(msg)
	}
	return string(password), err
}

type showAPIToken struct{}

func (c *showAPIToken) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "token-show",
		Usage:   "token-show",
		Desc:    "Show API token user. If him does not have a key, it is generated.",
		MinArgs: 0,
	}
}

func (c *showAPIToken) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/api-key")
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
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var APIKey string
		err = json.Unmarshal(b, &APIKey)
		if err != nil {
			return err
		}
		fmt.Fprintf(context.Stdout, "API key: %s\n", APIKey)
	}
	return nil
}

type regenerateAPIToken struct{}

func (c *regenerateAPIToken) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "token-regenerate",
		Usage:   "token-regenerate",
		Desc:    "Generates a new API key. If there is already a key, it is replaced.",
		MinArgs: 0,
	}
}

func (c *regenerateAPIToken) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/api-key")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var APIKey string
		err = json.Unmarshal(b, &APIKey)
		if err != nil {
			return err
		}
		fmt.Fprintf(context.Stdout, "Your new API key is: %s\n", APIKey)
	}
	return nil
}
