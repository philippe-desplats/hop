// Command hop is a fast project-directory switcher. The binary prints results;
// a generated shell function performs the actual cd (a child process cannot
// change its parent shell's working directory). See `hop init <shell>`.
package main

import (
	"fmt"
	"os"

	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "0.0.1-dev"

// command is one subcommand of the table dispatch is driven from. Keeping the
// handler, the help page and the machine-facing mark in one declaration is what
// lets a rule hold for every subcommand at once: a new entry inherits it instead
// of having to remember it.
type command struct {
	run func(args []string)
	// helpKey is the catalog key of this subcommand's own help page. Empty means
	// the general help, which is what all but setup and init print.
	helpKey string
	// machineFacing marks a subcommand whose stdout is a protocol rather than
	// prose, so a help page must never be written to it.
	machineFacing bool
}

// commands is the whole subcommand set. nav and complete are the two
// machine-facing ones: nav's stdout carries the __HOP_CD__ / __HOP_RUN__
// sentinel lines the generated shell function captures, and complete's stdout
// feeds the shell's completion engine. A help page there would be swallowed by
// the capture or offered as a candidate, so both keep answering an unknown
// option the way they do today, on stderr and with a non-zero exit.
var commands = map[string]command{
	"nav":      {run: cmdNav, machineFacing: true},
	"complete": {run: cmdComplete, machineFacing: true},
	"setup":    {run: cmdSetup, helpKey: "cli.help.setup"},
	"init":     {run: cmdInit, helpKey: "cli.help.init"},
	"query":    {run: cmdQuery},
	"scan":     {run: cmdScan},
	"add":      {run: cmdAdd},
	"prompt":   {run: cmdPrompt},
	"config":   {run: cmdConfig},
	"pin":      {run: cmdPin},
	"unpin":    {run: cmdUnpin},
	"import":   {run: cmdImport},
	"track":    {run: cmdTrack},
	"untrack":  {run: cmdUntrack},
	"clean":    {run: cmdClean},
	"doctor":   {run: cmdDoctor},
}

func main() {
	i18n.SetLanguage(core.LoadSettings().UI.Language)
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		return
	}
	if cmd, ok := commands[args[0]]; ok {
		rest := args[1:]
		// Answer a help request here, before the handler body: in a terminal
		// --help is the universal reflex, and it must never be the gesture that
		// writes a config, builds an index, edits an rc file or opens a wizard.
		if !cmd.machineFacing && wantsHelp(rest) {
			printCommandHelp(cmd.helpKey)
			return
		}
		cmd.run(rest)
		return
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Printf("hop %s\n", version)
	case "help", "--help", "-h":
		printHelp()
	default:
		// A name hop does not know is an error even when a help flag follows it:
		// `hop nope --help` is a typo to report, not a question to answer.
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.unknown_command", args[0]))
		os.Exit(2)
	}
}

// wantsHelp reports whether args asks for help. The test is on the exact token
// on purpose: each subcommand parses its own arguments, and everything that is
// not spelled --help or -h must still reach it, including nav's `-` and `-2`,
// init's `--cmd=NAME` and import's `--from`.
func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

// printCommandHelp writes a subcommand's help page, falling back to the general
// help for the subcommands that have no page of their own.
func printCommandHelp(helpKey string) {
	if helpKey == "" {
		printHelp()
		return
	}
	fmt.Print(i18n.T(helpKey))
}

func printHelp() {
	fmt.Print(i18n.T("cli.help"))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "hop: %v\n", err)
	os.Exit(1)
}
