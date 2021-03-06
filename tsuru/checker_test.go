// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/tsuru/tsuru/cmd"
	"launchpad.net/gocheck"
)

type deprecationChecker struct{}

func (deprecationChecker) Info() *gocheck.CheckerInfo {
	return &gocheck.CheckerInfo{
		Name:   "deprecates",
		Params: []string{"new-name", "old-name"},
	}
}

func (deprecationChecker) Check(params []interface{}, names []string) (bool, string) {
	if len(params) != 2 {
		return false, "two parameters are needed"
	}
	newName, ok := params[0].(string)
	if !ok {
		return false, "new-name should be a string"
	}
	oldName, ok := params[1].(string)
	if !ok {
		return false, "old-name should be a string"
	}
	manager := buildManager("tsuru")
	newCommand, ok := manager.Commands[newName]
	if !ok {
		return false, newName + " is not registered"
	}
	oldCommand, ok := manager.Commands[oldName]
	if !ok {
		return false, oldName + " is not registered"
	}
	deprecated, ok := oldCommand.(*cmd.DeprecatedCommand)
	if !ok {
		return false, oldName + " is not registered as deprecated"
	}
	return deprecated.Command == newCommand, ""
}
