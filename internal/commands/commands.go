package commands

import (
	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/commands/create"
	initcmd "github.com/sqve/grove/internal/commands/init"
	"github.com/sqve/grove/internal/commands/list"
)

func NewCreateCmd() *cobra.Command {
	app := create.NewApp()
	return create.NewCreateCmd(app)
}

func NewInitCmd() *cobra.Command {
	return initcmd.NewInitCmd()
}

func NewListCmd() *cobra.Command {
	return list.NewListCmd()
}
