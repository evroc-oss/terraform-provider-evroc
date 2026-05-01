// Copyright 2026 evroc
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"

	"github.com/evroc-oss/terraform-provider-evroc/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

// version is set via ldflags at build time
var version string = "dev"

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{
		ProviderFunc: provider.New(version),
		Debug:        debugMode,
	}

	plugin.Serve(opts)
}
