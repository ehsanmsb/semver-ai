package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/config"
	"io"
	"os"
)

var buildVersion = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "usage: release-manager <validate-config|version> [--config PATH] [--json]")
		return 2
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(errOut)
	path := flags.String("config", ".release-manager.yml", "configuration path")
	asJSON := flags.Bool("json", false, "JSON output")
	if e := flags.Parse(args[1:]); e != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return 2
	}
	if args[0] == "version" {
		fmt.Fprintln(out, buildVersion)
		return 0
	}
	if args[0] != "validate-config" {
		fmt.Fprintln(errOut, "unknown command")
		return 2
	}
	if _, e := config.Load(*path); e != nil {
		fmt.Fprintln(errOut, e)
		return 1
	}
	if *asJSON {
		if e := json.NewEncoder(out).Encode(map[string]string{"status": "VALID"}); e != nil {
			return 1
		}
	} else {
		fmt.Fprintln(out, "Configuration is valid")
	}
	return 0
}
