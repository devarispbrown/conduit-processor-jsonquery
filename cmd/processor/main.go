//go:build wasm

package main

import (
	sdk "github.com/conduitio/conduit-processor-sdk"
	jsonquery "github.com/devarispbrown/conduit-processor-jsonquery"
)

func main() {
	sdk.Run(jsonquery.NewProcessor())
}
