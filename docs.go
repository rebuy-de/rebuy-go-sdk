// Package sdk is a library for our Golang projects.
//
// Development Status: rebuy-go-sdk is designed for internal use. Since it
// uses Semantic Versioning (https://semver.org/) it is safe to use, but expect
// big changes between major version updates.
//
// # Application Layout
//
// ## General Directory Structure
//
// Please take a look at the examples directory to see how it actually looks like.
//
//	/
//	├── cmd/[subcommand/]
//	│   ├── root.go
//	│   └── ...
//	├── pkg/
//	│   ├── app/...
//	│   ├── dal/...
//	│   ├── bll/...
//	│   └── ...
//	├── buildutil
//	├── go.mod
//	├── go.sum
//	├── LICENSE
//	├── main.go
//	├── README.md
//	└── tools.go
//
// - /buildutil is a convenience wrapper to execute the buildutil command
// from the SDK. It ensures that the application gets built with a defined
// version of buildutil.
//
// - /main.go is the entrypoint of the application. It's typically very minimal,
// containing just enough code to initialize the command framework and handle errors.
// Its primary responsibility is to set up the application with the SDK's cmdutil package
// and delegate execution to the Cobra command structure defined in /cmd/root.go.
//
// - /tools.go forces dependency management of modules, that are not directly
// imported. This is important for go run and go generate that use external
// modules. See wiki for more details: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
//
// - /cmd/root.go contains the definition for all Cobra commands and
// the Runners (see below) of the application. This is where you define your command-line
// interface structure, options, and connect the commands to their implementations.
//
// ## Command Structure Organization
//
// The separation of concerns in command files follows a clear pattern:
//
// main.go - Minimal application entry point:
//
//	func main() {
//	   defer cmdutil.HandleExit()
//	
//	   if err := cmd.NewRootCommand().Execute(); err != nil {
//	       logrus.Fatal(err)
//	   }
//	}
//
// cmd/root.go - Command definition and runner setup:
//
//	func NewRootCommand() *cobra.Command {
//	   runner := new(Runner)
//	
//	   cmd := cmdutil.New(
//	       "myapp", "github.com/org/myapp",
//	       cmdutil.WithLogVerboseFlag(),
//	       cmdutil.WithVersionCommand(),
//	       cmdutil.WithRunner(runner),
//	   )
//	
//	   // Add additional subcommands if needed
//	   cmd.AddCommand(newSubCommand())
//	
//	   return cmd
//	}
//	
//	// Runner implementation follows...
//
// cmd/server.go - Server configuration and setup:
//
//	// RunServer configures and starts the application server with dependency injection
//	func RunServer(ctx context.Context, c *dig.Container) error {
//	   // Register core dependencies
//	   err := errors.Join(
//	       // Register template viewer
//	       c.Provide(func(templateFS fs.FS) *webutil.GoTemplateViewer {
//	           return webutil.NewGoTemplateViewer(templateFS,
//	               webutil.SimpleTemplateFuncMap("formatTime", FormatTimeFunction),
//	           )
//	       }),
//	
//	       // Register HTTP handlers
//	       webutil.ProvideHandler(c, handlers.NewUserHandler),
//	       webutil.ProvideHandler(c, handlers.NewDashboardHandler),
//	
//	       // Register background workers
//	       runutil.ProvideWorker(c, workers.NewSyncWorker),
//	
//	       // Register the HTTP server itself
//	       runutil.ProvideWorker(c, webutil.NewServer),
//	   )
//	   if err != nil {
//	       return err
//	   }
//	
//	   // Start all registered workers
//	   return runutil.RunProvidedWorkers(ctx, c)
//	}
//
// This separation of concerns follows a clear pattern:
//
// 1. /main.go initializes the command framework and handles errors
// 2. /cmd/root.go defines the CLI structure and environment-specific runners
// 3. /cmd/server.go contains shared server setup code used by all environments
//
// The environment-specific runners in root.go do initialization specific to their environment
// (production, development, etc.) and then call the common RunServer function to set up
// the application components that are environment-independent.
//
// - /pkg/app contains separate components of the application. The /pkg/app
// directory serves basically the same purpose as the /cmd, but is separated
// into multiple sub-packages. This is useful when the /cmd directory grows
// too big and contains components that are mostly independent from each other.
// How the sub packages of /pkg/app are designed is highly dependent on the
// application. It could be split model-based (eg users, projects, ...) or it
// could be split purpose-based (eg web, controllers, ...).
//
// - /pkg/bll stands for "business logic layer" and contains sub-packages that
// solve a specific use-case of the application.
//
// - /pkg/dal stands for "data access layer" and contains sub-packages that
// serve as a wrapper for external services and APIs. The idea of grouping such
// packages is to make their purpose clear and to avoid mixing access to
// external services with actual business logic.
//
// # Major Release Notes
//
// - vN is the new release (eg v3)
// - vP is the previous one (eg v2)
//
// 1. Create a new branch release-vN to avoid breaking changes getting into the previous release.
// 2. Do your breaking changes in the branch.
// 3. Update the imports everywhere:
//   * find . -type f -exec sed -i 's#github.com/rebuy-de/rebuy-go-sdk/vO#github.com/rebuy-de/rebuy-go-sdk/vP#g' {} +
// 4. Merge your branch.
// 5. Add Release on GitHub.
package sdk