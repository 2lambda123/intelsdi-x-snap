package main

import (
	"bufio"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/codegangsta/cli"
)

func loadPlugin(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		fmt.Print("Incorrect usage\n")
		os.Exit(1)
	}
	pb, err := readPlugin(ctx.Args().First())
	if err != nil {
		fmt.Printf("Error: %v\n", err.Error())
		os.Exit(1)
	}
	err = client.LoadPlugin(pb)
	if err != nil {
		fmt.Printf("Error: %v\n", err.Error())
		os.Exit(1)
	}
}

func listPlugins(ctx *cli.Context) {
	lps, aps, err := client.GetPlugins(ctx.Bool("running"))
	if err != nil {
		fmt.Printf("Error: %v\n", err.Error())
		os.Exit(1)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	if ctx.Bool("running") {
		printFields(w, false, 0, "NAME", "HIT COUNT", "LAST HIT", "TYPE")
		for _, rp := range aps {
			printFields(w, false, 0, rp.Name, rp.HitCount, rp.LastHit.Format(time.RFC1123), rp.TypeName)
		}
	} else {
		printFields(w, false, 0, "NAME", "STATUS", "LOADED TIMESTAMP")
		for _, lp := range lps {
			printFields(w, false, 0, lp.Name, lp.Status, lp.LoadedTimestamp)
		}
	}
	w.Flush()
}

func readPlugin(filename string) ([]byte, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats, statsErr := file.Stat()
	if statsErr != nil {
		return nil, statsErr
	}

	var size = stats.Size()
	bytes := make([]byte, size)

	bufr := bufio.NewReader(file)
	_, err = bufr.Read(bytes)

	return bytes, err
}
