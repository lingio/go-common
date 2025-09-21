# go-common

- Monitoring [common.InitMonitoring(svcName, monitorCfg)](trace.go#L56).
  - Traces: otel http sink, gcp cloud trace

- Debugging
  - go tool pprof

```
go install golang.org/x/tools/cmd/goimports@latest
```

## spanner

### setup local spanner

- requires gcloud ([install](https://cloud.google.com/sdk/docs/install))
  - `snap` is not supported, [uninstall](https://cloud.google.com/sdk/docs/downloads-snap) and use link above!
- requires gcloud configuration `emulator`
- requires spanner emulator component

Setting up `emulator` config:
```bash
# Setup gcloud configuration
gcloud config configurations create --activate emulator
gcloud config set auth/disable_credentials true
gcloud config set project lingio-test
gcloud config set api_endpoint_overrides/spanner http://localhost:9020/
```

Installing the local spanner emulator:
```bash
gcloud components update
# Install as needed when prompted.
# If gcloud is installed via google-cloud-cli, ensure apt install command starts with google-cloud-cli instead of google-cloud-sdk
gcloud emulators spanner start
```

Finally, copy staging database to local emulator.
```bash
cd ~/my-service/
bash ~/go-common/script/setup-spanner-emulator.sh gcp-project-id spanner-instance-id database-name
```

There are certain [limitations](https://cloud.google.com/spanner/docs/emulator#limitations).

### spanner-tools

Some useful commands that `wrench` and `spanner-cli` does not cover.

Especially, copying data from staging to local emulator is *much* faster than with `spanner-cli`.

This program is installed by setup-spanner-emulator.sh

```bash
go install github.com/lingio/go-common/script/spanner-tools@latest
spanner-tools -h
```

## storagegen

```bash
go install github.com/lingio/go-common/storagegen@latest
storagegen path/to/service/storage/spec.json
```

**spec.json**:
```javascript
{
  "serviceName": "person-service",
  "buckets": [
    {
      "typeName": "People",       // final type name: {typeName}Store
      "dbTypeName": "DbPerson",   // stored and returned type: models.{dbTypeName}
      "bucketName": "people",     // object store bucket name
      "version": "v1",            // change this if the stored data structure is changed
      "getAll": false,            // enable to generate code for listing all objects
      "filenameFormat": "%s.json" // fmt.Sprintf format, must have exactly one %s (ID)
      "secondaryIndexes": [       // generate additional Get methods
        // GetByPhone, builds index on models.{dbTypeName}.Phone
        { "key": "Phone", "type": "unique" },
        // GetByEmail, builds index on *models.{dbTypeName}.Email if it not nil
        { "keys": [{ "key": "Email", "optional": true }], "type": "unique" },
        // GetByPartnerAndEmail, builds a compound index on models.{dbTypeName}.Partner and *models.{dbTypeName}.Email if it not nil
        { "keys": [{ "key": "Partner" }, { "key": "Email", "optional": true }], "type": "unique", "name": "PartnerAndEmail" },
        // GetAllByPartnerAndStudentGroup
        {
          "keys": [
            { "key": "Partner" },
            { "key": "Student", "optional": true, "exclFromIndex": true },
            { "key": "Student.GroupID", "param": "groupID" }
          ],
          "type": "set",
          "name": "PartnerAndStudentGroup"
        },
        // GetAllByPartner, builds index on models.{dbTypeName}.Partner
        { "key": "Partner", "type": "set"}
      ],
      // directstore.tmpl: encrypted object store
      // cachedstore.tmpl: cache + encrypted object store
      // blobstore.tmpl: directstore for []byte data
      "template": "cachedstore.tmpl",
      "config": {
        // Defaults to "application/json". Will be applied on object Put.
        "contentType": "application/json",
        // Defaults to "". Will be applied on object Put.
        "contentDisposition": ""
      }
    }
  ]
}
```

## scripts

- `script/objcopy`: read/write objects from/to bucket from one s3 endpoint
  > `MINIO_SECRET=xyz go run ./script/objcopy --from=path/to/{stage,local,local-stage}.json --bucket=xyz`
- `script/encrypt`: encrypt or decrypt plaintext objects on stdin
  > `ENCRYPTION_KEY=256bitkey go run ./script/encrypt`
- `script/fromfile`: read filenames from stdin and write object to stdout
  > `find ./dir -not -type d | go run ./script/fromfile`
- `script/tofile`: read objects from stdin and write files
  > `... go run ./script/objcopy | go run ./script/tofile -root=./dir`
- `script/objify`: like `fromfile` but reads raw json objects
  > `cat data.jsonl | go run ./script/objify`

##### Write plaintext objects to disk from an encrypted object storage

```bash
# assuming encrypted bucket
$ MINIO_SECRET=minioadmin go run ./script/objcopy --from=../service/config/local-stage.json --bucket=people | \
ENCRYPTION_KEY=256bit-key go run ./script/encrypt --decrypt | \
go run ./script/tofile --root=./files --rename="people-data-{KEY}{EXT}"
```

##### Write encrypted objects to an encrypted object storage from disk

```bash
# assuming encrypted bucket
$ find ../files -maxdepth 1 -not -type d | gp run ./script/fromfile | \
ENCRYPTION_KEY=256bit-key go run ./script/encrypt | \
MINIO_SECRET=minioadmin go run ./script/objcopy --to=../service/config/local-stage.json --bucket=people
```
