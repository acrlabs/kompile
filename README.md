# kompile
A demo Kubernetes compiler

* Run `make build` to generate the `kompile` binary and the demo binary
* Run `.build/demo` to start the demo binary locally; then run `curl -XPOST localhost:8080/upload --data-binary
  @"me.jpg"` to have the image be resized.
* Run `.build/kompile -f demo/main.go` to generate the Kubernetes-compiled objects
