package main

import (
  "os"
  "io/ioutil"
  "fmt"
  "time"
  "net/http"
  "strings"
  "path/filepath"
  "github.com/gorhill/cronexpr"
  "encoding/json"
  "sort"
)

type ByName []Project

type Project struct {
  Name string
  Routes []*Route `json:"items"`
}

type Route struct {
  Metadata struct {
    Name string
    Namespace string
  }
  Spec struct {
    Host string
    Tls struct {
      Termination string
    }
  }
}

func (p ByName) Len() int {
    return len(p)
}
func (p ByName) Swap(i, j int) {
    p[i], p[j] = p[j], p[i]
}
func (p ByName) Less(i, j int) bool {
    return p[i].Name < p[j].Name
}

func renew(renewCronExpr *cronexpr.Expression) {
  
  for {
    now := time.Now()
    next := renewCronExpr.Next(now)
    fmt.Printf("Next certificate renewal run at %v\n", next)
    time.Sleep(next.Sub(now))
    
    routeFiles, _ := filepath.Glob("/var/lib/letsencrypt/*/*.json")
    for _, routeFile := range routeFiles {
      project := filepath.Base(filepath.Dir(routeFile))
      route :=  strings.TrimSuffix(filepath.Base(routeFile), filepath.Ext(routeFile))
      proc := sh("/usr/local/letsencrypt/bin/letsencrypt.sh '%s' '%s' `cat /run/secrets/kubernetes.io/serviceaccount/token`", project, route)
      fmt.Println(proc.stderr + proc.stdout)
    }
  }
}

func handler(w http.ResponseWriter, r *http.Request) {
  if r.Header["Authorization"] == nil {
    w.WriteHeader(http.StatusForbidden)
    return
  }

  token := strings.Split(r.Header["Authorization"][0], " ")[1]

  tempDir, err := ioutil.TempDir("/tmp", "letsencrypt")
  if err != nil {
    fmt.Fprintln(w, err)
    return
  }

  defer os.RemoveAll(tempDir)

  os.Setenv("TMPDIR", tempDir)
  os.Setenv("KUBECONFIG", tempDir + "/.kubeconfig")

  proc := sh("oc login kubernetes.default.svc.cluster.local:443 --certificate-authority=/run/secrets/kubernetes.io/serviceaccount/ca.crt --token=%s", token)
  if proc.Err() != nil {
    fmt.Fprintln(w, proc.Stderr() + proc.Stdout())
    return
  }

  if r.Method == http.MethodPost {
    handlePost(w, r, token)
  } else {
    handleGet(w, r, token)
  }
}

func handleGet(w http.ResponseWriter, r *http.Request, token string) {
  projectNames := strings.Split(sh("oc get project -o jsonpath='{.items[*].metadata.name}'").Stdout(), " ")
  projects := make([]Project, len(projectNames))
  for i, projectName := range projectNames {
    json.Unmarshal(sh("oc get routes -n %s -o json", projectName).StdoutBytes(), &projects[i])
    projects[i].Name = projectName
  }

  sort.Sort(ByName(projects))
  LetsencryptTmpl(w, projects)
}

func handlePost(w http.ResponseWriter, r *http.Request, token string) {
  project := r.FormValue("project")
  route := r.FormValue("route")

  proc := sh("/usr/local/letsencrypt/bin/letsencrypt.sh '%s' '%s' '%s' 2>&1", project, route, token)
  fmt.Fprintln(w, proc.Stdout())
}

func main() {

 renewCron := os.Getenv("RENEW_CRON")
  if renewCron == "" {
    renewCron = "@daily"
  }

  renewCronExpr := cronexpr.MustParse(renewCron)
  if renewCronExpr.Next(time.Now()).IsZero() {
    panic("Cron expression doesn't match any future dates!")
  }
 
  go renew(renewCronExpr)

  http.HandleFunc("/", handler)
  http.Handle("/.well-known/acme-challenge/", http.FileServer(http.Dir("/srv")))
  http.ListenAndServe(":8080", nil)
}
