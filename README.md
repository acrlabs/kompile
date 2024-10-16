# kompile
A demo Kubernetes compiler

* Run `make` to generate the `kompile` binary and the demo binary, build the demo Docker image, and deploy it to k8s
* Run `.build/demo` to start the demo binary locally; then run `curl -XPOST localhost:8080/upload --data-binary
  @"me.jpg"` to have the image be resized.
* Port-forward to the demo pod on 8080, and then run the above `curl` command to run the demo binary on k8s
* Run `.build/kompile -f demo/main.go` to generate the Kubernetes-compiled objects
