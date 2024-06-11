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
