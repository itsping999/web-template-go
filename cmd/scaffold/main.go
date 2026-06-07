package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/wyuhsin/web-template-go/internal/scaffold"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "proto":
		if err := runProto(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "scaffold proto: %v\n", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runProto(args []string) error {
	fs := flag.NewFlagSet("proto", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "project root")
	targetDir := fs.String("target-dir", "internal/service", "service implementation target directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: go run ./cmd/scaffold proto <package>/<version>/<name>.proto")
	}

	result, err := scaffold.RunProto(context.Background(), scaffold.ProtoOptions{
		RootDir:          *root,
		Proto:            fs.Arg(0),
		ServiceTargetDir: *targetDir,
	})
	if err != nil {
		return err
	}

	fmt.Printf("created proto: %s\n", result.ProtoPath)
	fmt.Printf("created service: %s\n", result.ServicePath)
	if len(result.NextSteps) > 0 {
		fmt.Println("next steps:")
		for _, step := range result.NextSteps {
			fmt.Printf("- %s\n", step)
		}
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  go run ./cmd/scaffold proto <package>/<version>/<name>.proto

Examples:
  go run ./cmd/scaffold proto order/v1/order.proto
  make scaffold-proto PROTO=order/v1/order.proto

The proto workflow wraps Kratos CLI generation, fixes go_package for this repo,
runs make api, formats the generated service stub, and prints the next wiring
steps for this template.`)
}
