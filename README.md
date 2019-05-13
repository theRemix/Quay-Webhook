## Quay webhook

### Config

see [example.hcl](./example.hcl)

### Build

```sh
make
```

### Run

```sh
./bin/webhook example.hcl
```

### Test

see [test-master-payload.json](./test-master-payload.json) and [test-tag-payload.json](./test-tag-payload.json)

```sh
curl -X POST --data-binary "@./test-master-payload.json" localhost:2000
curl -X POST --data-binary "@./test-tag-payload.json" localhost:2000
```
