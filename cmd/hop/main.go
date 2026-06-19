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

func main() {
	i18n.SetLanguage(core.LoadSettings().UI.Language)
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		return
	}
	switch args[0] {
	case "nav":
		cmdNav(args[1:])
	case "query":
		cmdQuery(args[1:])
	case "scan":
		cmdScan(args[1:])
	case "add":
		cmdAdd(args[1:])
	case "init":
		cmdInit(args[1:])
	case "setup":
		cmdSetup(args[1:])
	case "prompt":
		cmdPrompt(args[1:])
	case "config":
		cmdConfig(args[1:])
	case "complete":
		cmdComplete(args[1:])
	case "pin":
		cmdPin(args[1:])
	case "unpin":
		cmdUnpin(args[1:])
	case "import":
		cmdImport(args[1:])
	case "track":
		cmdTrack(args[1:])
	case "untrack":
		cmdUntrack(args[1:])
	case "clean":
		cmdClean(args[1:])
	case "doctor":
		cmdDoctor(args[1:])
	case "version", "--version", "-v":
		fmt.Printf("hop %s\n", version)
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "hop: commande inconnue %q (essaie `hop help`)\n", args[0])
		os.Exit(2)
	}
}

func printHelp() {
	fmt.Print(i18n.T("cli.help"))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "hop: %v\n", err)
	os.Exit(1)
}
