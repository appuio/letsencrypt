package main

import "fmt"
import "strings"
import "os/exec"
import "bytes"

type proc struct {
  stdout string
  stderr string
  err error
  checkedErrors bool
}

func (proc *proc) CheckErrors() {
  if proc.checkedErrors && proc.err != nil {
    panic(proc.err.Error() + "\n" + proc.stderr)
  }
}

func (proc *proc) Stdout() string {
  proc.CheckErrors()
  return proc.stdout
}

func (proc *proc) StdoutLines() []string {
  proc.CheckErrors()
  return strings.Split(proc.stdout, "\n")
}

func (proc *proc) Err() error {
  proc.checkedErrors = true
  return proc.err
}

func sh(cmd string, a ...interface{}) *proc {
  var command *exec.Cmd
  if len(a) == 0 {
    command = exec.Command("bash", "-c", cmd)
  } else {
    command = exec.Command("bash", "-c", fmt.Sprintf(cmd, a...))
  }
  var stderr bytes.Buffer
  command.Stderr = &stderr
  out, err := command.Output()
  
  return &proc { 
    stdout: strings.TrimSuffix(string(out), "\n"),
    stderr: strings.TrimSuffix(stderr.String(), "\n"),
    err: err,
  }
}
