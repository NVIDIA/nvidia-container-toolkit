# Welcome to `urfave/cli-altsrc/v3`

[![Run Tests](https://github.com/urfave/cli-altsrc/actions/workflows/main.yml/badge.svg)](https://github.com/urfave/cli-altsrc/actions/workflows/main.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/urfave/cli-altsrc/v3.svg)](https://pkg.go.dev/github.com/urfave/cli-altsrc/v3)
[![Go Report Card](https://goreportcard.com/badge/github.com/urfave/cli-altsrc/v3)](https://goreportcard.com/report/github.com/urfave/cli-altsrc/v3)

[`urfave/cli-altsrc/v3`](https://pkg.go.dev/github.com/urfave/cli-altsrc/v3) is an extension for [`urfave/cli/v3`] to read
flag values from JSON, YAML, and TOML. The extension keeps third-party libraries for these features away from [`urfave/cli/v3`].

[`urfave/cli/v3`]: https://github.com/urfave/cli

### Example

```go
	configFiles := []string{
		filepath.Join(testdataDir, "config.yaml"),
		filepath.Join(testdataDir, "alt-config.yaml"),
	}

	app := &cli.Command{
		Name: "greet",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Sources: yaml.YAML("greet.name", configFiles...),
			},
			&cli.IntFlag{
				Name:    "enthusiasm",
				Aliases: []string{"!"},
				Sources: yaml.YAML("greet.enthusiasm", configFiles...),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			punct := ""
			if cmd.Int("enthusiasm") > 9000 {
				punct = "!"
			}

			fmt.Fprintf(os.Stdout, "Hello, %[1]v%[2]v\n", cmd.String("name"), punct)

			return nil
		},
	}

	// Simulating os.Args
	os.Args = []string{"greet"}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stdout, "OH NO: %[1]v\n", err)
	}

	// Output:
	// Hello, Berry!
```
