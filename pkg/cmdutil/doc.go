// Package cmdutil contains helper utilities for setting up a CLI with Go,
// providing basic application behavior and for reducing boilerplate code.
//
// An example application can be found at
// https://github.com/rebuy-de/golang-template.
//
// # Graceful Application Exits
//
// In many command line applications it is desired to exit the process
// immediately, if it is clear that the application cannot recover. Important
// note: This is designed for actual applications (ie not libraries), because
// only the application itself should decide when to exit. Libraries should
// alway return errors.
//
// There are three ways to handle fatal errors in Go. With os.Exit() the
// process will terminate immediately, but it will not call any deferrers which
// means that possible cleanup task do not get called. The next way is to call
// panic, which respects the defer statements, but unfortunately it is not
// possible to define an exit code and the user gets confused with a stack
// trace. Finally, the function could just return an error indicating that
// things failed, but this introduces a lot of code, conditionals and appears
// unnecessary, when it is already clear that the application cannot recover.
//
// The package cmdutil provides an alternative, which panics with a known
// struct and catches it right before the application exit. This is an example
// to illustrate the usage:
//
//	func main() {
//	  defer cmdutil.HandleExit()
//	  run()
//	}
//
//	func run() {
//	  defer fmt.Println("important cleanup")
//	  err := doSomething()
//	  if err != nil {
//	    log.Error(err)
//	    cmdutil.Exit(2)
//	  }
//	}
//
// The defer of HandleExit is the first statement in the main function. It
// ensures a pretty output and that the application exits with the specified
// exit code. The run function does something and makes the application exit
// with an exit code. The specified defer statement is still called. Also the
// application logging facility should be used to communicate the error, so the
// error actually appears on external logging applications like Syslog or
// Graylog.
//
// # Minimal Application Boilerplate
//
// Golang is very helpful for creating glue code in the ops area and creating
// micro services. But when you want features like proper logging, a version
// subcommand and a clean structure, there is still a lot of boilerplate code
// needed. NewRootCommand creates a ready-to-use Cobra command to reduce the
// necessary code. This is an example to illustrate the usage:
//
//	type App struct {
//	    Name string
//	}
//
//	func (app *App) Run(cmd *cobra.Command, args []string) {
//	    log.Infof("hello %s", app.Name)
//	}
//
//	func (app *App) Bind(cmd *cobra.Command) {
//	    cmd.PersistentFlags().StringVarP(
//	        &app.Name, "name", "n", "world",
//	        `Your name.`)
//	}
//
//	func NewRootCommand() *cobra.Command {
//	    cmd := cmdutil.NewRootCommand(new(App))
//	    cmd.Short = "an example app for golang which can be used as template"
//	    return cmd
//	}
//
// The App struct contains fields for parameters which are defined in Bind or
// for internal states which might get defined while running the application.
//
// # Command Structure with cmdutil
//
// The SDK provides a streamlined approach to creating command-line applications. Here's how to set up your application:
//
//	func main() {
//	    defer cmdutil.HandleExit()
//	
//	    cmd := cmdutil.New(
//	        "myapp",                          // Short app name
//	        "github.com/org/myapp",           // Full app name
//	        cmdutil.WithLogVerboseFlag(),     // Add -v flag for verbose logging
//	        cmdutil.WithLogToGraylog(),       // Add Graylog support
//	        cmdutil.WithVersionCommand(),     // Add version command
//	        cmdutil.WithVersionLog(logrus.DebugLevel),
//	        cmdutil.WithRunner(new(Runner)),  // Add main application runner
//	    )
//	
//	    if err := cmd.Execute(); err != nil {
//	        logrus.Fatal(err)
//	    }
//	}
//
// This approach provides a consistent interface for command-line applications with built-in support for logging, versioning, and other common capabilities.
//
// # Runner Pattern
//
// Runners are structs that define command line flags and prepare the application for launch.
//
// ## Basic Runner Structure
//
//	type Runner struct {
//	    name string
//	    redisAddress string
//	    // Other configuration fields
//	}
//
//	// Bind defines command line flags
//	func (r *Runner) Bind(cmd *cobra.Command) error {
//	    cmd.PersistentFlags().StringVar(
//	        &r.name, "name", "World",
//	        `Your name.`)
//
//	    cmd.PersistentFlags().StringVar(
//	        &r.redisAddress, "redis-address", "localhost:6379",
//	        `Redis server address.`)
//
//	    return nil
//	}
//
//	// Run executes the main application logic
//	func (r *Runner) Run(ctx context.Context) error {
//	    // Application setup and launch
//	    return nil
//	}
//
// ## Environment-Specific Runners
//
// You can create different environment configurations for your application:
//
//	// Run for production environment
//	func (r *Runner) Run(ctx context.Context) error {
//	    redisClient := redis.NewClient(&redis.Options{
//	        Addr: r.redisAddress,
//	    })
//
//	    // Production setup
//	    return r.runServer(ctx, redisClient)
//	}
//
//	// Dev runs the server in development mode
//	func (r *Runner) Dev(ctx context.Context, cmd *cobra.Command, args []string) error {
//	    // Create a local test Redis instance
//	    podman, err := podutil.DevPodman(ctx)
//	    if err != nil {
//	        return err
//	    }
//
//	    keydbContainer, err := podutil.StartDevcontainer(ctx, podman, "app-dev-keydb",
//	        "docker.io/eqalpha/keydb:latest")
//	    if err != nil {
//	        return err
//	    }
//
//	    redisClient := redis.NewClient(&redis.Options{
//	        Addr: keydbContainer.TCPHostPort(6379),
//	    })
//
//	    // Development setup with hot reloading, etc.
//	    return r.runServer(ctx, redisClient)
//	}
//
//	// Shared server setup with environment-specific dependencies
//	func (r *Runner) runServer(ctx context.Context, redisClient *redis.Client) error {
//	    // Common server setup and run
//	}
//
// The purpose of splitting the Runner and the actual application code is:
// - to get initializing errors as fast as possible (eg if the Redis server is not available),
// - to be able to execute environment-specific code without having to use conditionals all over the code-base,
// - to be able to mock services for local development
// - and to define a proper interface for the application launch, which is very helpful for e2e tests.
//
// # Version Command
//
// NewRootCommand also attaches NewVersionCommand to the application. It prints
// the compiled version of the application and other build parameters. These
// values need to be set by the build system via ldflags.
//
//	BUILD_XDST=$(pwd)/vendor/github.com/rebuy-de/rebuy-go-sdk/cmdutil
//	go build -ldflags "\
//	  -X '${BUILD_XDST}.BuildName=${NAME}' \
//	  -X '${BUILD_XDST}.BuildPackage=${PACKAGE}' \
//	  -X '${BUILD_XDST}.BuildVersion=${BUILD_VERSION}' \
//	  -X '${BUILD_XDST}.BuildDate=${BUILD_DATE}' \
//	  -X '${BUILD_XDST}.BuildHash=${BUILD_HASH}' \
//	  -X '${BUILD_XDST}.BuildEnvironment=${BUILD_ENVIRONMENT}' \
package cmdutil
