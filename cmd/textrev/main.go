package main

import (
	"flag"
	"fmt"
	"os"

	"example.com/text-revision-plan/internal/textrev"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "textrev:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("textrev", flag.ContinueOnError)
	planPath := fs.String("plan", "", "path to the JSON revision plan")
	srcDir := fs.String("src", "", "source directory containing original UTF-8 text files")
	outDir := fs.String("out", "", "directory to write revised files into (must not exist)")
	check := fs.Bool("check", false, "validate only; create no output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *planPath == "" || *srcDir == "" {
		return fmt.Errorf("usage: textrev -plan plan.json -src <dir> [-out <dir>] [-check]")
	}
	if !*check && *outDir == "" {
		return fmt.Errorf("-out is required unless -check is set")
	}
	summary, err := textrev.Run(*planPath, *srcDir, *outDir, *check)
	if err != nil {
		return err
	}
	data, err := textrev.MarshalSummary(summary)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
