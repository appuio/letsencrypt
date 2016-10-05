package internal

type Routes struct {
  Items []*Route
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
