// Copyright 2014 tsuru authors. All rights reserved.
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
	"sort"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"launchpad.net/gnuflag"
)

type serviceList struct{}

func (s serviceList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-list",
		Usage: "service-list",
		Desc:  "Get all available services, and user's instances for this services",
	}
}

func (s serviceList) Run(ctx *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/services/instances")
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	rslt, err := cmd.ShowServicesInstancesList(b)
	if err != nil {
		return err
	}
	n, err := ctx.Stdout.Write(rslt)
	if n != len(rslt) {
		return errors.New("Failed to write the output of the command")
	}
	return nil
}

type serviceAdd struct {
	fs        *gnuflag.FlagSet
	teamOwner string
}

func (c *serviceAdd) Info() *cmd.Info {
	usage := `service-add <servicename> <serviceinstancename> [plan] [-t/--owner-team <team>]
e.g.:

    $ tsuru service-add mongodb tsuru_mongodb small -t myteam

Will add a new instance of the "mongodb" service, named "tsuru_mongodb" with the plan "small".`
	return &cmd.Info{
		Name:    "service-add",
		Usage:   usage,
		Desc:    "Create a service instance to one or more apps make use of.",
		MinArgs: 2,
		MaxArgs: 3,
	}
}

func (c *serviceAdd) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName, instanceName := ctx.Args[0], ctx.Args[1]
	var plan string
	if len(ctx.Args) > 2 {
		plan = ctx.Args[2]
	}
	var b bytes.Buffer
	params := map[string]string{
		"name":         instanceName,
		"service_name": serviceName,
		"plan":         plan,
		"owner":        c.teamOwner,
	}
	err := json.NewEncoder(&b).Encode(params)
	if err != nil {
		return err
	}
	url, err := cmd.GetURL("/services/instances")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Service successfully added.\n")
	return nil
}

func (c *serviceAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		flagDesc := "the team that owns te service (mandatory if the user is member of more than one team)"
		c.fs = gnuflag.NewFlagSet("service-add", gnuflag.ExitOnError)
		c.fs.StringVar(&c.teamOwner, "team-owner", "", flagDesc)
		c.fs.StringVar(&c.teamOwner, "t", "", flagDesc)
	}
	return c.fs
}

type serviceBind struct {
	cmd.GuessingCommand
}

func (sb *serviceBind) Run(ctx *cmd.Context, client *cmd.Client) error {
	appName, err := sb.Guess()
	if err != nil {
		return err
	}
	instanceName := ctx.Args[0]
	url, err := cmd.GetURL("/services/instances/" + instanceName + "/" + appName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	w := tsuruIo.NewStreamWriter(ctx.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, resp.Body) {
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

func (sb *serviceBind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-bind",
		Usage: "service-bind <instancename> [-a/--app appname]",
		Desc: `bind a service instance to an app

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
}

type serviceUnbind struct {
	cmd.GuessingCommand
}

func (su *serviceUnbind) Run(ctx *cmd.Context, client *cmd.Client) error {
	appName, err := su.Guess()
	if err != nil {
		return err
	}
	instanceName := ctx.Args[0]
	url, err := cmd.GetURL("/services/instances/" + instanceName + "/" + appName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	w := tsuruIo.NewStreamWriter(ctx.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, resp.Body) {
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

func (su *serviceUnbind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-unbind",
		Usage: "service-unbind <instancename> [-a/--app appname]",
		Desc: `unbind a service instance from an app

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
}

type serviceInstanceStatus struct{}

func (c serviceInstanceStatus) Info() *cmd.Info {
	usg := `service-status <serviceinstancename>
e.g.:

    $ tsuru service-status my_mongodb
`
	return &cmd.Info{
		Name:    "service-status",
		Usage:   usg,
		Desc:    "Check status of a given service instance.",
		MinArgs: 1,
	}
}

func (c serviceInstanceStatus) Run(ctx *cmd.Context, client *cmd.Client) error {
	instName := ctx.Args[0]
	url, err := cmd.GetURL("/services/instances/" + instName + "/status")
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
	bMsg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	msg := string(bMsg) + "\n"
	n, err := fmt.Fprint(ctx.Stdout, msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return errors.New("Failed to write to standard output.\n")
	}
	return nil
}

type serviceInfo struct{}

func (c serviceInfo) Info() *cmd.Info {
	usg := `service-info <service>
e.g.:

    $ tsuru service-info mongodb
`
	return &cmd.Info{
		Name:    "service-info",
		Usage:   usg,
		Desc:    "List all instances of a service",
		MinArgs: 1,
	}
}

type ServiceInstanceModel struct {
	Name string
	Apps []string
	Info map[string]string
}

// in returns true if the list contains the value
func in(value string, list []string) bool {
	for _, item := range list {
		if value == item {
			return true
		}
	}
	return false
}

func (serviceInfo) ExtraHeaders(instances []ServiceInstanceModel) []string {
	var headers []string
	for _, instance := range instances {
		for key := range instance.Info {
			if !in(key, headers) {
				headers = append(headers, key)
			}
		}
	}
	sort.Sort(sort.StringSlice(headers))
	return headers
}

func (c serviceInfo) BuildInstancesTable(serviceName string, ctx *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/services/" + serviceName)
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
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var instances []ServiceInstanceModel
	err = json.Unmarshal(result, &instances)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte(fmt.Sprintf("Info for \"%s\"\n\n", serviceName)))
	if len(instances) > 0 {
		ctx.Stdout.Write([]byte("Instances\n"))
		table := cmd.NewTable()
		extraHeaders := c.ExtraHeaders(instances)
		for _, instance := range instances {
			apps := strings.Join(instance.Apps, ", ")
			data := []string{instance.Name, apps}
			for _, h := range extraHeaders {
				data = append(data, instance.Info[h])
			}
			table.AddRow(cmd.Row(data))
		}
		headers := []string{"Instances", "Apps"}
		headers = append(headers, extraHeaders...)
		table.Headers = cmd.Row(headers)
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

func (c serviceInfo) BuildPlansTable(serviceName string, ctx *cmd.Context, client *cmd.Client) error {
	ctx.Stdout.Write([]byte("\nPlans\n"))
	url, err := cmd.GetURL(fmt.Sprintf("/services/%s/plans", serviceName))
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
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var plans []map[string]string
	err = json.Unmarshal(result, &plans)
	if err != nil {
		return err
	}
	if len(plans) > 0 {
		table := cmd.NewTable()
		for _, plan := range plans {
			data := []string{plan["Name"], plan["Description"]}
			table.AddRow(cmd.Row(data))
		}
		table.Headers = cmd.Row([]string{"Name", "Description"})
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

func (c serviceInfo) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	err := c.BuildInstancesTable(serviceName, ctx, client)
	if err != nil {
		return err
	}
	return c.BuildPlansTable(serviceName, ctx, client)
}

type serviceDoc struct{}

func (serviceDoc) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-doc",
		Usage:   "service-doc <servicename>",
		Desc:    "Show documentation of a service",
		MinArgs: 1,
	}
}

func (serviceDoc) Run(ctx *cmd.Context, client *cmd.Client) error {
	sName := ctx.Args[0]
	url := fmt.Sprintf("/services/%s/doc", sName)
	url, err := cmd.GetURL(url)
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
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	ctx.Stdout.Write(result)
	return nil
}

type serviceRemove struct {
	yes bool
	fs  *gnuflag.FlagSet
}

func (c *serviceRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-remove",
		Usage:   "service-remove <serviceinstancename> [--assume-yes]",
		Desc:    "Removes a service instance",
		MinArgs: 1,
	}
}

func (c *serviceRemove) Run(ctx *cmd.Context, client *cmd.Client) error {
	name := ctx.Args[0]
	var answer string
	if !c.yes {
		fmt.Fprintf(ctx.Stdout, `Are you sure you want to remove service "%s"? (y/n) `, name)
		fmt.Fscanf(ctx.Stdin, "%s", &answer)
		if answer != "y" {
			fmt.Fprintln(ctx.Stdout, "Abort.")
			return nil
		}
	}
	url := fmt.Sprintf("/services/instances/%s", name)
	url, err := cmd.GetURL(url)
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
	fmt.Fprintf(ctx.Stdout, `Service "%s" successfully removed!`+"\n", name)
	return nil
}

func (c *serviceRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("service-remove", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.yes, "assume-yes", false, "Don't ask for confirmation, just remove the service.")
		c.fs.BoolVar(&c.yes, "y", false, "Don't ask for confirmation, just remove the service.")
	}
	return c.fs
}
