/*
Copyright © 2022 GMO Pepabo, inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/goccy/go-json"
	"github.com/pepabo/alive-arns/arn"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var format string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "alive-arns",
	Short: "print alive AWS Resource Names",
	Long:  `print alive AWS Resource Names.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return err
		}

		regions, err := regions(ctx, cfg)
		if err != nil {
			return err
		}

		c := arn.NewCollector()
		arns := arn.Arns{}
		for _, r := range regions {
			cfg.Region = r
			log.Info().Msg(fmt.Sprintf("Checking %s", r))
			a, err := c.CollectArns(ctx, cfg)
			if err != nil {
				return err
			}
			arns = append(arns, a...)
		}

		switch format {
		case "json":
			e := json.NewEncoder(os.Stdout)
			e.SetIndent("", "  ")
			if err := e.EncodeContext(ctx, arns.Unique().Sort()); err != nil {
				return err
			}
		default:
			for _, a := range arns.Unique().Sort() {
				cmd.Printf("%s\n", a)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&format, "format", "t", "", "output format")
}

func Execute() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("DEBUG") != "" && os.Getenv("DEBUG") != "0" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func regions(ctx context.Context, cfg aws.Config) ([]string, error) {
	ec2s := ec2.NewFromConfig(cfg)
	rs, err := ec2s.DescribeRegions(ctx, &ec2.DescribeRegionsInput{AllRegions: aws.Bool(false)})
	if err != nil {
		return nil, err
	}
	regions := []string{}
	for _, r := range rs.Regions {
		regions = append(regions, *r.RegionName)
	}
	return regions, nil
}
