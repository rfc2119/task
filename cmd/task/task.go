package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	shellquote "github.com/kballard/go-shellquote"
	"github.com/spf13/pflag"
	"mvdan.cc/sh/v3/syntax"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/args"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

var (
	version = ""
)

const usage = `Usage: task [-ilfwvsd] [--init] [--list] [--force] [--watch] [--verbose] [--silent] [--dir] [--taskfile] [--dry] [--summary] [task...]

Runs the specified task(s). Runs a built-in shell if no task name
was specified, or lists all tasks if an unknown task name was specified.

Example: 'task hello' with the following 'Taskfile.yml' file will generate an
'output.txt' file with the content "hello".

'''
version: '3'
tasks:
  hello:
    cmds:
      - echo "I am going to write a file named 'output.txt' now."
      - echo "hello" > output.txt
    generates:
      - output.txt
'''

Options:
`

// repl provides a bare REPL functionality to Task
func repl() error {

	log.Println("Type 'help' for a list of commands or 'quit' to exit ")
	rl, err := readline.New("task> ")
	if err != nil {
		return err
	}
	defer rl.Close()
REPL:
	for {

		// TODO: support context signals and autocompletion
		line, err := rl.Readline()
		if err != nil {
			log.Fatalf("readline error: %s", err)
			break
		}
		args, err := shellquote.Split(line)
		if err != nil {
			log.Printf("Error: %s", err)
			continue
		}
		if len(args) < 1 {
			continue
		}
		switch args[0] {
		case "quit":
			break REPL // long live goto!
		case "clear":
			readline.ClearScreen(os.Stderr)
			continue
		case "help":
			pflag.Usage()
			continue
		default:
			// log.Printf("Unknown command %s", args[0])
			// continue
		}

		os.Args = append([]string{"task"}, args...)
		start(true)

	}
	return nil

}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	pflag.Usage = func() {
		log.Print(usage)
		pflag.PrintDefaults()
	}

	// Launches a shell-like interface if no arguments were provided
	if len(os.Args) == 1 {
		if err := repl(); err != nil {
			log.Printf("%v", err)
			os.Exit(1)
		}
		return
	}
	start(false)
}

// start is the entry function for Task
func start(calledFromRepl bool) {

	var (
		versionFlag bool
		helpFlag    bool
		init        bool
		list        bool
		status      bool
		force       bool
		watch       bool
		verbose     bool
		silent      bool
		dry         bool
		summary     bool
		parallel    bool
		concurrency int
		dir         string
		entrypoint  string
		output      string
		color       bool
	)

	// Reset the internal state for the pflag package. This is necessary
	// to prevent redefining flags (which causes pflag to panic).
	// This is typically relevant in testing input with command line flags
	// as seen in tools like Cobra, kubectl, calico, ... etc.
	// Instead of reseting the internal state of pflag, you could operate
	// on a new set of flags:
	// 	newFlags := pflag.NewFlagSet(args[0], ContinueOnError)
	//	...
	//	newFlags.Parse(args[1:])
	if calledFromRepl {
		pflag.CommandLine = pflag.NewFlagSet("task", pflag.PanicOnError)
	}
	pflag.BoolVar(&versionFlag, "version", false, "show Task version")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "shows Task usage")
	pflag.BoolVarP(&init, "init", "i", false, "creates a new Taskfile.yml in the current folder")
	pflag.BoolVarP(&list, "list", "l", false, "lists tasks with description of current Taskfile")
	pflag.BoolVar(&status, "status", false, "exits with non-zero exit code if any of the given tasks is not up-to-date")
	pflag.BoolVarP(&force, "force", "f", false, "forces execution even when the task is up-to-date")
	pflag.BoolVarP(&watch, "watch", "w", false, "enables watch of the given task")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "enables verbose mode")
	pflag.BoolVarP(&silent, "silent", "s", false, "disables echoing")
	pflag.BoolVarP(&parallel, "parallel", "p", false, "executes tasks provided on command line in parallel")
	pflag.BoolVar(&dry, "dry", false, "compiles and prints tasks in the order that they would be run, without executing them")
	pflag.BoolVar(&summary, "summary", false, "show summary about a task")
	pflag.StringVarP(&dir, "dir", "d", "", "sets directory of execution")
	pflag.StringVarP(&entrypoint, "taskfile", "t", "", `choose which Taskfile to run. Defaults to "Taskfile.yml"`)
	pflag.StringVarP(&output, "output", "o", "", "sets output style: [interleaved|group|prefixed]")
	pflag.BoolVarP(&color, "color", "c", true, "colored output. Enabled by default. Set flag to false or use NO_COLOR=1 to disable")
	pflag.IntVarP(&concurrency, "concurrency", "C", 0, "limit number tasks to run concurrently")

	pflag.Parse()

	if versionFlag {
		fmt.Printf("Task version: %s\n", getVersion())
		return
	}

	if helpFlag {
		pflag.Usage()
		return
	}

	if init {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		if err := task.InitTaskfile(os.Stdout, wd); err != nil {
			log.Fatal(err)
		}
		return
	}

	if dir != "" && entrypoint != "" {
		log.Fatal("task: You can't set both --dir and --taskfile")
		return
	}
	if entrypoint != "" {
		dir = filepath.Dir(entrypoint)
		entrypoint = filepath.Base(entrypoint)
	} else {
		entrypoint = "Taskfile.yml"
	}

	e := task.Executor{
		Force:       force,
		Watch:       watch,
		Verbose:     verbose,
		Silent:      silent,
		Dir:         dir,
		Dry:         dry,
		Entrypoint:  entrypoint,
		Summary:     summary,
		Parallel:    parallel,
		Color:       color,
		Concurrency: concurrency,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,

		OutputStyle: output,
	}
	if err := e.Setup(); err != nil {
		log.Fatal(err)
	}
	v, err := e.Taskfile.ParsedVersion()
	if err != nil {
		log.Fatal(err)
		return
	}

	if list {
		e.PrintTasksHelp()
		return
	}

	var (
		calls   []taskfile.Call
		globals *taskfile.Vars
	)

	tasksAndVars, cliArgs, err := getArgs()
	if err != nil {
		log.Fatal(err)
	}

	if v >= 3.0 {
		calls, globals = args.ParseV3(tasksAndVars...)
	} else {
		calls, globals = args.ParseV2(tasksAndVars...)
	}

	globals.Set("CLI_ARGS", taskfile.Var{Static: cliArgs})
	e.Taskfile.Vars.Merge(globals)

	ctx := context.Background()
	if !watch {
		ctx = getSignalContext()
	}

	if status {
		if err := e.Status(ctx, calls...); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := e.Run(ctx, calls...); err != nil {
		e.Logger.Errf(logger.Red, "%v", err)
		if !calledFromRepl {
			os.Exit(1)
		}
	}
}

func getArgs() ([]string, string, error) {
	var (
		args          = pflag.Args()
		doubleDashPos = pflag.CommandLine.ArgsLenAtDash()
	)

	if doubleDashPos == -1 {
		return args, "", nil
	}

	var quotedCliArgs []string
	for _, arg := range args[doubleDashPos:] {
		quotedCliArg, err := syntax.Quote(arg, syntax.LangBash)
		if err != nil {
			return nil, "", err
		}
		quotedCliArgs = append(quotedCliArgs, quotedCliArg)
	}
	return args[:doubleDashPos], strings.Join(quotedCliArgs, " "), nil
}

func getSignalContext() context.Context {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-ch
		log.Printf("task: signal received: %s", sig)
		cancel()
	}()
	return ctx
}

func getVersion() string {
	if version != "" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "unknown"
	}

	version = info.Main.Version
	if info.Main.Sum != "" {
		version += fmt.Sprintf(" (%s)", info.Main.Sum)
	}

	return version
}
