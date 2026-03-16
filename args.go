package main

import (
	"flag"
	"strings"
)

// parseFlags reorders args so flags come before positionals, enabling
// interleaved usage like: add name version file --namespace foo
// Returns positionals via fs.Args() after parsing.
func parseFlags(fs *flag.FlagSet, args []string) error {
	var flagArgs, posArgs []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			// If next arg is a value (not another flag), consume it
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			posArgs = append(posArgs, args[i])
		}
	}
	// Put flags first so flag.FlagSet sees them before positionals
	return fs.Parse(append(flagArgs, posArgs...))
}
