To run the e2e test you should place yourself in  `.../test/e2e` directory, then

```shell
$go test -c
```
which will compile the `e2e.test` executable in your current directory. Then

```shell
./e2e.test --kubeconfig=$HOME/.kube/config

```

will start the e2e test....
