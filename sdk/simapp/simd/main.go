package main

import (
	"fmt"
	"os"

	"github.com/rollkit/cosmos-sdk-starter/sdk/simapp"
	"github.com/rollkit/cosmos-sdk-starter/sdk/simapp/simd/cmd"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", simapp.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
