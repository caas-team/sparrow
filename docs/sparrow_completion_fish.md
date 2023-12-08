## sparrow completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	sparrow completion fish | source

To load completions for every new session, execute once:

	sparrow completion fish > ~/.config/fish/completions/sparrow.fish

You will need to start a new shell for this setup to take effect.


```
sparrow completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sparrow.yaml)
```

### SEE ALSO

* [sparrow completion](sparrow_completion.md)	 - Generate the autocompletion script for the specified shell

