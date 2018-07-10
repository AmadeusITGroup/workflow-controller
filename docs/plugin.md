# Workflow kubectl plugin

```console
> kubectl plugin workflow
This plugin show Custom Resources information handled by Workflow-Controller".

Usage:
  workflow [command]

Available Commands:
  cronworkflow shows cronworkflow custom resources
  help         Help about any command
  workflow     shows workflow custom resources

Flags:
      --config string   config file (default is $HOME/.kubectl-plugin.yaml)
  -h, --help            help for workflow
  -t, --toggle          Help message for toggle

Use "workflow [command] --help" for more information about a command.

```

## Example of display evolution with a workflow resource

```console
> kubectl plugin workflow workflow
NAME            NAMESPACE  STATUS   STEPS  CURRENT STEP(S)  JOB(S) STATUS
hello-workflow  default    Created  0/2    -                -

> kubectl plugin workflow workflow
NAME            NAMESPACE  STATUS   STEPS  CURRENT STEP(S)  JOB(S) STATUS
hello-workflow  default    Running  1/2    one              wfl-hello-workflow-one-kjxfc(Created)

> kubectl plugin workflow workflow
NAME            NAMESPACE  STATUS   STEPS  CURRENT STEP  CORRESPONDING JOB
hello-workflow  default    Running  2/2    two              wfl-hello-workflow-one-kjxfc(Complete),wfl-hello-workflow-two-rj5x6(Running)

> kubectl plugin workflow workflow
NAME            NAMESPACE  STATUS    STEPS  CURRENT STEP(S)  JOB(S) STATUS
hello-workflow  default    Running   2/2    two              wfl-hello-workflow-one-kjxfc(Complete),wfl-hello-workflow-two-rj5x6(Running)

> kubectl plugin workflow workflow
NAME            NAMESPACE  STATUS    STEPS  CURRENT STEP(S)  JOB(S) STATUS
hello-workflow  default    Complete  2/2    -                wfl-hello-workflow-one-kjxfc(Complete),wfl-hello-workflow-two-rj5x6(Complete)
```