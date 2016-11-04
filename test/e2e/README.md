To run the e2e test you should place yourself in  `.../test/e2e` directory, then

```shell
$go test -c
```
which will compile the `e2e.test` executable in your current directory. Then

```shell
./e2e.test --host=192.168.10.1:8080 --kubeconfig=$HOME/.kube/config --group=example.com --name=workflow --version=v1

```

will start the e2e test....
