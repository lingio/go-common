go-common

## redis cache

```golang
// Basic client: connect to redis addr directly
// redisAddr = ":6769", masterName "", serviceDNS = ""
// Failover client: lookup sentinels and create failover client
// redisAddr = "", masterName != "", serviceDNS != ""
redisClient, err := common.SetupRedisClient(redisAddr, masterName, serviceDNS)
if err != nil {
  log.Fatalln(err)
}
redisCache := common.NewRedisCache(redisClient, "x-cache-it", "v2")
```

## storagegen

1. `cd storagegen`
2. `go run . ../../partner-service/storage/spec.json`

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
        // GetAllByPartner, builds index on models.{dbTypeName}.Partner
        { "key": "Partner", "type": "set"}
      ],
      // directstore.tmpl: encrypted object store
      // minio1.tmpl: cache + encrypted object store
      "template": "minio1.tmpl",
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
  > `MINIO_SECRET=xyz go run ./script/objcopy --from=path/to/{stage|local|local-stage}.json --bucket=xyz`
- `script/encrypt`: encrypt or decrypt plaintext objects on stdin
  > `ENCRYPTION_KEY=256bitkey go run ./script/encrypt`
- `script/fromfile`: read filenames from stdin and write object to stdout
- `script/tofile`: read objects from stdin and write files

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
