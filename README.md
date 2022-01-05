# Run

Run is a fork of [Task](https://github.com/go-task/task)

## New Features

### Prompt

```yaml
# A prompt field provides an interactive method of getting user data
# In addition, it controls execution with the `validate` sub-field
# which, on correct input, executes the task `answer`
# The user's selection (or input) is available as the {{.ANSWER}} template variable
  test_prompt:
    vars:
      TEST: "string"
    prompt:
      # type: select
      # type: multi_select
      # type: multiline
      type: input
      # type: password
      # type: confirm
      message: What day is it?
# The options sub-field is used with propmpts of type select and multi_select
      # options:
      #   - msg: Sunday
      #   - Tuesday
      # The following is a dynamic option
      #   - msg:
      #       sh: date +%A
      validate:
        sh: '[[ "{{.ANSWER}}" == "Tuesday" ]]'
      answer:
        desc: "a task executed on valid input only"
        cmds:
          - echo "successfully executing the answer task"
```

### Initial Shell Script
```yaml
version: '3'

# shell_rc is used to load common shell scripts or functions
# A global shell_rc
shell_rc: |
    func(){
      echo "global function called!"
    }

tasks:

  init-script-global:
    desc: Testing a local shell_rc field
    cmds:
    - echo "trying out the global shell_rc field"
    - func

  init-script:
    desc: Testing a local shell_rc field
    shell_rc: |
      export VAR_INSIDE_INIT_SCRIPT="Hello from init script"
      func(){
        echo "local function called!"
      }
    cmds:
      - echo "This is a var inside init_script $VAR_INSIDE_INIT_SCRIPT"
      - func
      - sleep 2   # doing some work
```
### Hide tasks from being listed
```yaml

# The `hide` field allows a task to not be listed with task --list
# It can be also templated using Go templates
# NOTE: 
#   remember that Go is strongly typed. comparison must be done between equal types
#   use double quotes for literals inside templates

  error-with-hide:
    desc: A hidden task that exits with an errors
    vars:
      CGO_ENABLED: "true"
    hide: '{{if eq .CGO_ENABLED "true"}} true {{else}} false {{end}}'
    # hide: true
    cmds:
      - echo "text"
      - exit 1
      - echo "unreachable"
  ```
### Initial Status
```yaml
# The initial_status field allows a task  to be executed 
# once if the `status` has been successfully executed once
tasks:
  default:
    cmds:
      - generate-files
      - rm -rf directory/
      - generate-files

  generate-files:
    desc: Generate files diescription
    cmds:
      - mkdir directory
      - touch directory/file1.txt
      - touch directory/file2.txt
    # test existence of files
    status:
      - test -d directory
      - test -f directory/file1.txt
      - test -f directory/file2.txt
    initial_status: true

```
On running `task default` from the command line, only the first execution of task `generate-files` is done
### Aliases
From the command line, you may call a task by its alias instead of its name
```yaml
  echo-with-errors-ignored:
    desc: Echoes a string but with errors ignored
    # Try calling the task from the command line as `task hello`
    alias: hello
    cmds:
      - cmd: exit 1
        ignore_error: true
      - echo "Hello World"
```

### More output messages
running `task` with more `-v`s produces more verbose output

| Option   | Effect    |
|--------------- | --------------- |
| `-v`   | Output each command executed, task running time   |
| `-vv`   | Output execution time of each command |


### Interactive Prompt
Executing `task` spawns a REPL-like shell by default. If you would like to execute the default task, please do `task default`
```
user@user:$ task
Type 'help' for a list of commands or 'quit' to exit 
task> --list

   Tasks                                                                      
                                                                              
              TASK           │ ALIAS │          DESCRIPTION                   
  ───────────────────────────┼───────┼─────────────────────────────────       
    echo-with-errors-ignored │ hello │ Echoes a string but with               
                             │       │ errors ignored                         
    generate-files           │       │ Generate files diescription            
    init-script              │       │ Testing the new shell_rc field         
    simple                   │       │ A simple task with no extra            
                             │       │ features                               
    sleep                    │       │ zzzzzzz                                
    test_prompt              │       │ tests prompt                           
    test_prompt_confirm      │       │ test prompt confirm                    
    test_prompt_password     │       │ test prompt password                   

task> simple
task: [simple] echo "Hello"
Hello
task> sleep
task: [sleep] sleep "2"
task> ^D
readline error: EOF
user@user:$ 
```

### Fancy listing
`task --list` uses the [list](https://github.com/charmbracelet/bubbles#list) component from bubbletea

[![bubbletea_list_demo](https://asciinema.org/a/sem2Ac3yZIUJ03HTMHyOEOq7I)](https://asciinema.org/a/sem2Ac3yZIUJ03HTMHyOEOq7I)

### [WIP] Progress bar
Trach the issue [here](https://github.com/charmbracelet/bubbletea/issues/179)
