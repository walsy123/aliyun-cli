// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/i18n"
)

type Command struct {
	// Command Name
	Name string

	// Short is the short description shown in the 'help' output.
	Short *i18n.Text

	// Long is the long message shown in the 'help <this-command>' output.
	Long *i18n.Text

	// Syntax for usage
	Usage string

	// Sample command
	Sample string

	// Enable unknown flags
	EnableUnknownFlag bool

	// enable suggest distance,
	// disable -1
	// 0: default distance
	SuggestDistance int

	// Hidden command
	Hidden bool

	// Run, command error will be catch
	Run func(ctx *Context, args []string) error

	// Help
	Help func(ctx *Context, args []string) error

	// auto compete
	AutoComplete func(ctx *Context, args []string) []string

	parent      *Command
	subCommands []*Command
	flags       *FlagSet

	// Keep args
	KeepArgs bool
}

func (c *Command) AddSubCommand(cmd *Command) {
	cmd.parent = c
	c.subCommands = append(c.subCommands, cmd)
}

func (c *Command) Flags() *FlagSet {
	if c.flags == nil {
		c.flags = NewFlagSet()
	}
	return c.flags
}

func (c *Command) Execute(ctx *Context, args []string) {
	if ctx.completion != nil {
		args = ctx.completion.GetArgs()
	}

	err := c.executeInner(ctx, args)
	if err != nil {
		c.processError(ctx, err)
	}
}

func (c *Command) getName() string {
	if c.parent == nil {
		return c.Name
	}

	return c.parent.getName() + " " + c.Name
}

type Metadata struct {
	Name   string                   `json:"name"`
	Short  map[string]string        `json:"short"`
	Long   map[string]string        `json:"long"`
	Usage  string                   `json:"usage"`
	Sample string                   `json:"sample"`
	Hidden bool                     `json:"hidden"`
	Flags  map[string]*MetadataFlag `json:"flags"`
}

type MetadataFlag struct {
	Name         string            `json:"name"`
	Shorthand    rune              `json:"shorthand"`
	Short        map[string]string `json:"short"`
	Long         map[string]string `json:"long"`
	DefaultValue string            `json:"default"`
	Required     bool              `json:"required"`
	Aliases      []string          `json:"aliases"`
	AssignedMode int               `json:"assign_mode"`
	Persistent   bool              `json:"persistent"`
	Hidden       bool              `json:"hidden"`
	Category     string            `json:"category"`
}

func (c *Command) GetMetadata(metadata map[string]*Metadata) {
	name := c.getName()

	meta := &Metadata{}
	meta.Name = name
	meta.Short = c.Short.GetData()
	if c.Long != nil {
		meta.Long = c.Long.GetData()
	}

	meta.Usage = c.Usage
	meta.Sample = c.Sample
	meta.Hidden = c.Hidden

	meta.Flags = make(map[string]*MetadataFlag)
	for _, flag := range c.Flags().Flags() {
		f := &MetadataFlag{}
		f.Name = flag.Name
		f.Shorthand = flag.Shorthand
		if flag.Short != nil {
			f.Short = flag.Short.GetData()
		}
		if flag.Long != nil {
			f.Long = flag.Long.GetData()
		}
		f.DefaultValue = flag.DefaultValue
		f.Required = flag.Required
		f.Aliases = flag.Aliases
		f.AssignedMode = int(flag.AssignedMode)
		f.Persistent = flag.Persistent
		f.Hidden = flag.Hidden
		f.Category = flag.Category

		// Flag can assigned with --flag field1=value1 field2=value2 value3 ...
		// must used with AssignedMode=AssignedRepeatable
		// Fields []Field

		// Flag can't appear with other flags, use Flag.Name
		// ExcludeWith []string

		meta.Flags[flag.Name] = f
	}
	metadata[name] = meta
	for _, cmd := range c.subCommands {
		cmd.GetMetadata(metadata)
	}
}

func (c *Command) GetSubCommand(s string) *Command {
	for _, cmd := range c.subCommands {
		if cmd.Name == s {
			return cmd
		}
	}
	return nil
}

func (c *Command) GetSuggestions(s string) []string {
	sr := NewSuggester(s, c.GetSuggestDistance())
	for _, cmd := range c.subCommands {
		sr.Apply(cmd.Name)
	}
	return sr.GetResults()
}

func (c *Command) GetSuggestDistance() int {
	if c.SuggestDistance < 0 {
		return 0
	} else if c.SuggestDistance == 0 {
		return DefaultSuggestDistance
	} else {
		return c.SuggestDistance
	}
}

func (c *Command) GetUsageWithParent() string {
	usage := c.Usage
	for p := c.parent; p != nil; p = p.parent {
		usage = p.Name + " " + usage
	}
	return usage
}

func (c *Command) ExecuteComplete(ctx *Context, args []string) {
	if strings.HasPrefix(ctx.completion.Current, "-") {
		for _, f := range ctx.flags.Flags() {
			if f.Hidden {
				continue
			}
			if !strings.HasPrefix("--"+f.Name, ctx.completion.Current) {
				continue
			}
			Printf(ctx.Stdout(), "--%s\n", f.Name)
		}
	} else {
		for _, sc := range c.subCommands {
			if sc.Hidden {
				continue
			}
			if !strings.HasPrefix(sc.Name, ctx.completion.Current) {
				continue
			}
			Printf(ctx.Stdout(), "%s\n", sc.Name)
		}
	}
}

func (c *Command) executeInner(ctx *Context, args []string) error {
	// fmt.Printf(">>> Execute Command: %s args=%v\n", c.Name, args)
	parser := NewParser(args, ctx)

	var current = parser.GetCurrent()
	// get next arg
	nextArg, _, err := parser.ReadNextArg()
	if err != nil {
		return err
	}

	// if next arg is help, run help
	if nextArg == "help" {
		ctx.help = true
		return c.executeInner(ctx, parser.GetRemains())
	}

	// if next args is not empty, try find sub commands
	if nextArg != "" {
		// if has sub command, run it
		subCommand := c.GetSubCommand(nextArg)
		if subCommand != nil {
			ctx.EnterCommand(subCommand)
			return subCommand.executeInner(ctx, parser.GetRemains())
		}

		// no sub command and command.Run == nil
		// raise error
		if c.Run == nil {
			// c.executeHelp(ctx, args, fmt.Errorf("unknown command: %s", nextArg))
			return NewInvalidCommandError(nextArg, ctx)
		}
	}

	var remainArgs []string
	if !c.KeepArgs {
		// cmd is find by args, try run cmd.Run
		// parse remain args
		remainArgs, err = parser.ReadAll()
		if err != nil {
			return fmt.Errorf("parse failed %s", err)
		}
	} else {
		remainArgs = args[current:]
	}

	// check flags
	err = ctx.CheckFlags()
	if err != nil {
		return err
	}

	if HelpFlag(ctx.Flags()).IsAssigned() {
		ctx.help = true
	}

	callArgs := make([]string, 0)
	if nextArg != "" {
		if !c.KeepArgs {
			callArgs = append(callArgs, nextArg)
		}
	}

	for _, s := range remainArgs {
		if s != "help" {
			callArgs = append(callArgs, s)
		} else {
			ctx.help = true
		}
	}

	if ctx.completion != nil {
		if c.AutoComplete != nil {
			ss := c.AutoComplete(ctx, callArgs)
			for _, s := range ss {
				Printf(ctx.Stdout(), "%s\n", s)
			}
		} else {
			c.ExecuteComplete(ctx, callArgs)
		}
		return nil
	}

	if ctx.help {
		c.executeHelp(ctx, callArgs)
		return nil
	} else if c.Run == nil {
		c.executeHelp(ctx, callArgs)
		return nil
	}

	return c.Run(ctx, callArgs)
}

func (c *Command) processError(ctx *Context, err error) {
	Errorf(ctx.Stderr(), "ERROR: %s\n", err.Error())
	if e, ok := err.(SuggestibleError); ok {
		PrintSuggestions(ctx, i18n.GetLanguage(), e.GetSuggestions())
		Exit(2)
		return
	}
	if e, ok := err.(ErrorWithTip); ok {
		Noticef(ctx.Stderr(), "\n%s\n", e.GetTip(i18n.GetLanguage()))
		Exit(3)
		return
	}
	Exit(1)
}

func (c *Command) executeHelp(ctx *Context, args []string) {
	if c.Help != nil {
		err := c.Help(ctx, args)
		if err != nil {
			c.processError(ctx, err)
		}
		return
	}

	c.PrintHead(ctx)
	c.PrintUsage(ctx)
	c.PrintSubCommands(ctx)
	c.PrintFlags(ctx)
	c.PrintTail(ctx)
}
