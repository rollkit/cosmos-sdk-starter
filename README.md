# cosmos-sdk-starter

Provides a start command handler for rollkit which can be used by the cosmos-sdk apps

## Usage

### Requirements

* Go version >= 1.21
* Ignite CLI >= v28.3.0

### Steps

* Create a cosmos-sdk app using ignite cli `ignite scaffold chain gm --address-prefix gm`
* Add cosmos-sdk-starter to your `gm` project
  * `cd gm`
  * `go get github.com/rollkit/cosmos-sdk-starter`
  * `go mod tidy`
* Make sure to check that cosmos-sdk version is `v0.50.6+` and rollkit version is `v0.13.1+`
* Navigate to `cmd/gmd/cmd/commands.go` under your `gm` project
* Add following imports

  ```go
  rollserv "github.com/rollkit/cosmos-sdk-starter/server"
  rollconf "github.com/rollkit/rollkit/config"
  ```

* Edit `initRootCmd` function to replace

  ```go
  server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)
  ```

  to

  ```go
  server.AddCommandsWithStartCmdOptions(
  		rootCmd,
  		app.DefaultNodeHome,
  		newApp, appExport,
  		server.StartCmdOptions{
  			AddFlags:            rollconf.AddFlags,
  			StartCommandHandler: rollserv.StartHandler[servertypes.Application],
  		},
  )
  ```

* Build your `gm` chain using `ignite chain build`
* Your `gm` app is now using Rollkit instead of Cometbft

* For running the `gm` chain using Rollkit, it is important to add the Rollkit sequencer to `gm` app's `genesis.json` file. Follow instructions provided in the [adding rollkit sequencer to genesis](https://rollkit.dev/guides/create-genesis#_9-configuring-the-genesis-file)
* Finally lauch app by passing rollkit flags: e.g., `gmd start --rollkit.aggregator --rpc.laddr tcp://127.0.0.1:36657 --grpc.address 127.0.0.1:9290 --p2p.laddr "0.0.0.0:36656" --minimum-gas-prices="0.025stake" --rollkit.da_address "http://localhost:7980"`
