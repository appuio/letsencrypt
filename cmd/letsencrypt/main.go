package main

import (
  "os"
  "io/ioutil"
  "fmt"
  "net/http"
  "strings"
  . "github.com/appuio/letsencrypt/internal"
  "encoding/json"
)

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
    fmt.Fprintln(w,proc.Stderr() + proc.Stdout())
    return
  }

  projects := strings.Split(sh("oc get project -o jsonpath='{.items[*].metadata.name}'").Stdout(), " ")
  for _, project := range projects {
    var routes Routes
    json.Unmarshal(sh("oc get routes -n %s -o json", project).StdoutBytes(), &routes)
    for _, route := range routes.Items {
//      var termination string
//      if route.Spec.Tls != nil {
//        termination = route.Spec.Tls.Termination
  //    }
      fmt.Fprintln(w, route.Metadata.Name + " " + route.Spec.Host + " " + route.Spec.Tls.Termination)
    }
  }

//  proc := sh("/usr/local/letsencrypt/letsencrypt.sh '%s' '%s' 2>&1", strings.Split(r.URL.Path, "/")[1], strings.Split(r.Header["Authorization"][0], " ")[1])
//  fmt.Fprintln(w,proc.stderr + proc.stdout)    
}

func main() {

//  json.Unmarshal(Sh("oc get is -o json --all-namespaces").StdoutBytes(), &imageStreams)
  http.HandleFunc("/", handler)
  http.Handle("/.well-known/acme-challenge/", http.FileServer(http.Dir("/srv")))
  http.ListenAndServe(":8080", nil)
}
