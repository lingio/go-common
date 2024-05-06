package spanner

import (
	"context"
	"encoding/json"
	"time"
)

type (
	ModType string

	ChangeStreamRecord struct {
		TableName           string
		CommitTimestamp     time.Time
		ServerTransactionID string `json:"serverTransactionId"`
		ModType             ModType
	}
)

const (
	ModTypeInsert ModType = "INSERT"
	ModTypeUpdate ModType = "UPDATE"
	ModTypeDelete ModType = "DELETE"
)

// ChangeStreamHandler returns a change stream handler that converts spanner
// change stream records with primary key type K, calling the provided
// handler function with the record data and key.
func ChangeStreamHandler[K any](
	handler func(context.Context, ChangeStreamRecord, K) error,
) func(context.Context, *ReadResult) error {
	return func(ctx context.Context, result *ReadResult) error {
		for _, result := range result.ChangeRecords {
			for _, dcr := range result.DataChangeRecords {
				for _, m := range dcr.Mods {
					if m.Keys.IsNull() {
						continue
					}

					data, err := m.Keys.MarshalJSON()
					if err != nil {
						return err
					}
					var key K
					if err := json.Unmarshal(data, &key); err != nil {
						return err
					}

					csr := ChangeStreamRecord{
						TableName:           dcr.TableName,
						CommitTimestamp:     dcr.CommitTimestamp,
						ServerTransactionID: dcr.ServerTransactionID,
						ModType:             ModType(dcr.ModType),
					}

					if err := handler(ctx, csr, key); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}
