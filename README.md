# set-value
 
## test

arguments

```
kpt fn source data --fn-config ./data/arg-fn-config.yaml | go run main.go
```


```
kpt fn source data --fn-config ./data/env-fn-config.yaml | go run main.go
```

```
kpt fn eval ./data --fn-config ./data/arg-fn-config.yaml --image henderiw/set-value:latest
```
