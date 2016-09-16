package main

import (
  "os"
  "fmt"
  "time"
  "net/http"
  "strings"
  "path/filepath"
  "github.com/gorhill/cronexpr"
)

func renew(renewCronExpr *cronexpr.Expression) {
  
  for {
    now := time.Now()
    next := renewCronExpr.Next(now)
    fmt.Printf("Next certificate renewal run at %v\n", next)
    time.Sleep(next.Sub(now))
    
    certs, _ := filepath.Glob("/var/lib/letsencrypt/*.crt")
    for _, cert := range certs {
      domain :=  strings.TrimSuffix(filepath.Base(cert), filepath.Ext(cert))
      proc := sh("/usr/local/letsencrypt/bin/letsencrypt.sh '%s' `cat /run/secrets/kubernetes.io/serviceaccount/token`", domain)
      fmt.Println(proc.stderr + proc.stdout)
    }
  }
}

func handler(w http.ResponseWriter, r *http.Request) {
  if r.Header["Authorization"] != nil {
    proc := sh("/usr/local/letsencrypt/bin/letsencrypt.sh '%s' '%s' 2>&1", strings.Split(r.URL.Path, "/")[1], strings.Split(r.Header["Authorization"][0], " ")[1])
    fmt.Fprintln(w,proc.stderr + proc.stdout)    
  } else {
    w.WriteHeader(http.StatusForbidden)
  }  
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

//  now := time.Now()
//  next := cronexpr.MustParse("46 17 * * *").Next(now)
//  fmt.Println(next.Sub(now))
//  time.Sleep(next.Sub(now))
//  renew()

//  time.Sleep(3000 * time.Second)
  http.HandleFunc("/", handler)
  http.Handle("/.well-known/acme-challenge/", http.FileServer(http.Dir("/srv")))
  http.ListenAndServe(":8080", nil)
}
