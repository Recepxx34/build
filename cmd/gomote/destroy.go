// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/build/buildlet"
	"golang.org/x/build/internal/gomote/protos"
)

func legacyDestroy(args []string) error {
	if activeGroup != nil {
		return fmt.Errorf("command does not support groups")
	}

	fs := flag.NewFlagSet("destroy", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "destroy usage: gomote destroy <instance>")
		fs.PrintDefaults()
		if fs.NArg() == 0 {
			// List buildlets that you might want to destroy.
			cc, err := buildlet.NewCoordinatorClientFromFlags()
			if err != nil {
				log.Fatal(err)
			}
			rbs, err := cc.RemoteBuildlets()
			if err != nil {
				log.Fatal(err)
			}
			if len(rbs) > 0 {
				fmt.Printf("possible instances:\n")
				for _, rb := range rbs {
					fmt.Printf("\t%s\n", rb.Name)
				}
			}
		}
		os.Exit(1)
	}

	fs.Parse(args)
	if fs.NArg() != 1 {
		fs.Usage()
	}
	name := fs.Arg(0)
	bc, err := remoteClient(name)
	if err != nil {
		return err
	}
	return bc.Close()
}

func destroy(args []string) error {
	fs := flag.NewFlagSet("destroy", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "destroy usage: gomote destroy [instance]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Destroys a single instance, or all instances in a group.")
		fmt.Fprintln(os.Stderr, "Instance argument is optional with a group.")
		fs.PrintDefaults()
		if fs.NArg() == 0 {
			// List buildlets that you might want to destroy.
			client := gomoteServerClient(context.Background())
			resp, err := client.ListInstances(context.Background(), &protos.ListInstancesRequest{})
			if err != nil {
				log.Fatalf("unable to list possible instances to destroy: %s", statusFromError(err))
			}
			if len(resp.GetInstances()) > 0 {
				fmt.Printf("possible instances:\n")
				for _, inst := range resp.GetInstances() {
					fmt.Printf("\t%s\n", inst.GetGomoteId())
				}
			}
		}
		os.Exit(1)
	}

	fs.Parse(args)

	var destroySet []string
	if fs.NArg() == 1 {
		destroySet = append(destroySet, fs.Arg(0))
	} else if activeGroup != nil {
		for _, inst := range activeGroup.Instances {
			destroySet = append(destroySet, inst)
		}
	} else {
		fs.Usage()
	}
	for _, name := range destroySet {
		fmt.Fprintf(os.Stderr, "# Destroying %s\n", name)
		ctx := context.Background()
		client := gomoteServerClient(ctx)
		if _, err := client.DestroyInstance(ctx, &protos.DestroyInstanceRequest{
			GomoteId: name,
		}); err != nil {
			return fmt.Errorf("unable to destroy instance: %s", statusFromError(err))
		}
	}
	if activeGroup != nil {
		activeGroup.Instances = nil
		if err := storeGroup(activeGroup); err != nil {
			return err
		}
	}
	return nil
}
