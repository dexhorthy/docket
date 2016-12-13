package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/horthy/docket/allocations"
	"github.com/horthy/docket/client"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type CLI struct {
	cmd  *cobra.Command
	args []string
}

func NewCli(cmd *cobra.Command, args []string) *CLI {
	return &CLI{
		cmd:  cmd,
		args: args,
	}
}

// TODO server needs to handle Not Found better here
func (cli *CLI) Delete() error {
	host, err := cli.cmd.Flags().GetString("host")
	if err != nil {
		return err
	}

	if len(cli.args) != 1 {
		return errors.New("name is required")
	}
	name := cli.args[0]

	err = client.NewClient(host).Delete(name)
	if err != nil {
		return err
	}

	color.Green("Deleted %v", name)
	return nil
}

func (cli *CLI) Get() error {
	host, err := cli.cmd.Flags().GetString("host")
	if err != nil {
		return err
	}

	if len(cli.args) != 1 {
		return errors.New("name is required")
	}
	name := cli.args[0]

	allocation, err := client.NewClient(host).Get(name)
	if err != nil {
		return err
	}

	raw, err := json.MarshalIndent(*allocation, "", "    ")
	if err != nil {
		return err
	}
	fmt.Print(string(raw))
	return nil
}

func (cli *CLI) List() error {
	host, err := cli.cmd.Flags().GetString("host")
	if err != nil {
		return err
	}
	allocations, err := client.NewClient(host).List()
	if err != nil {
		return err
	}

	bytes, _ := json.MarshalIndent(allocations, "", "    ")
	fmt.Print(string(bytes))
	return nil
}

func (cli *CLI) Push() error {
	host, err := cli.cmd.Flags().GetString("host")
	if err != nil {
		return err
	}

	var file string
	if len(cli.args) != 1 {
		file = "docket.yml"
	}
	file = cli.args[0]

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	specs := []*allocations.AllocationSpecification{}

	err = yaml.Unmarshal(data, &specs)
	if err != nil {
		return err
	}

	theClient := client.NewClient(host)
	for _, allocation := range specs {
		created, err := theClient.CreateOrUpdate(allocation)
		if err != nil {
			return err
		}

		if created {
			color.Green("Created allocation %v", allocation.Name)
		} else {
			color.Green("Updated allocation %v", allocation.Name)
		}
	}

	return nil

}
