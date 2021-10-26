go-common

## storagegen

1. `cd storagegen`
2. `go run . ../../partner-service/storage/spec.json`

**spec.json**:
```json
{
  "serviceName": "person-service",
  "buckets": [
    {
      "typeName": "People",     // final type name: {typeName}Store
      "dbTypeName": "DbPerson", // stored and returned type: models.{dbTypeName}
      "bucketName": "people",   // object store bucket name
      "version": "v1",          // change this if the stored data structure is changed
      "secondaryIndexes": [     // generate additional Get methods
      	// GetByPhone, builds index on models.{dbTypeName}.Phone
        { "key": "Phone", "type": "unique" },
        // GetByEmail, builds index on *models.{dbTypeName}.Email if it not nil
        { "key": "Email", "type": "unique", "optional": true },
        // GetAllByPartner, builds index on models.{dbTypeName}.Partner
        { "key": "Partner", "type": "set"}
      ],
      // directstore.tmpl: encrypted object store
      // minio1.tmpl: cache + encrypted object store
      "template": "minio1.tmpl",
    }
  ]
}
```
