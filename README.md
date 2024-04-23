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

  ```
  rollserv "github.com/rollkit/cosmos-sdk-starter/server"
  rollconf "github.com/rollkit/rollkit/config"
  ```

* Edit `initRootCmd` function to replace

  ```
  server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)
  ```

  to

  ```
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
