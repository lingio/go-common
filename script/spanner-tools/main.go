package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"cloud.google.com/go/spanner"
)

var (
	tableName string
	project   string
	instance  string
	database  string
	limitRows int
)

type row struct {
	Table string
	Cols  []string
	Vals  []spanner.GenericColumnValue
}

func main() {
	flag.StringVar(&tableName, "t", "", "use only this table, empty/default implies all tables")
	flag.StringVar(&project, "p", "", "project id (required)")
	flag.StringVar(&instance, "i", "", "instance id (required)")
	flag.StringVar(&database, "d", "", "database id (required)")
	flag.IntVar(&limitRows, "limit", 100, "row limit")
	flag.Usage = func() {
		fmt.Printf("Usage of %s: \n", os.Args[0])
		fmt.Println(`
Run common dev operations against Google Spanner.

  spanner-tools [OPTIONS] CMD

CMD:

  row list   - read rows from table/tables, output to stdout in jsonl
  row insert - read rows from jsonl and insert into table/tables
  table list - list all tables sorted in parent-child order

OPTIONS:
			`)
		flag.PrintDefaults()
		fmt.Println(`
EXAMPES:

  # Fetch up 10 rows from the Users table
  spanner-tools -p my-project -i test -d main -t Users --limit 10 row list

  # List tables in database main
  spanner-tools -p my-project -i test -d main table list

  # Copy 10 rows from Users in my-project/test/main to emulated spanner test/test/main
  spanner-tools -p my-project -i test -d main -t Users --limit 10 row list > rows.jsonl
  SPANNER_EMULATOR_HOST=localhost:9010 spanner-tools -p test -i test -d main row insert < rows.jsonl
`)
	}
	flag.Parse()

	cli, err := spanner.NewClient(context.TODO(), fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database))
	if err != nil {
		log.Fatalln(err)
	}

	group := flag.Arg(0)
	switch group {
	case "row", "rows":
		switch flag.Arg(1) {
		case "list":
			enc := json.NewEncoder(os.Stdout)
			if err := copyDatabaseTo(cli, enc); err != nil {
				log.Fatalln(err)
			}
		case "insert":
			dec := json.NewDecoder(os.Stdin)
			if err := insertIntoDatabaseFrom(cli, dec); err != nil {
				log.Fatalln(err)
			}
		default:
			fmt.Println("row list")
			fmt.Println("row insert")
			os.Exit(1)
		}
	case "table":
		switch flag.Arg(1) {
		case "list":
			for _, t := range databaseTableNames(cli) {
				fmt.Println(t)
			}
		default:
			fmt.Println("table list")
			os.Exit(1)
		}

	default:
		log.Fatalln("unknown command", group)
	}
}

func copyDatabaseTo(cli *spanner.Client, enc interface{ Encode(t any) error }) error {
	var tablesToCopy []string
	if tableName != "" {
		tablesToCopy = append(tablesToCopy, tableName)
	} else {
		tablesToCopy = databaseTableNames(cli)
	}

	limit := strconv.Itoa(limitRows)
	for _, table := range tablesToCopy {
		log.Printf("SELECT * FROM %s LIMIT %s\n", table, limit)

		it := cli.ReadOnlyTransaction().Query(
			context.TODO(),
			spanner.NewStatement("SELECT * FROM "+table+" LIMIT "+limit),
		)
		if err := it.Do(func(r *spanner.Row) error {
			cols := r.ColumnNames()
			var vals []spanner.GenericColumnValue
			for _, col := range cols {
				var v spanner.GenericColumnValue
				if err := r.ColumnByName(col, &v); err != nil {
					return err
				}
				vals = append(vals, v)
			}
			if err := enc.Encode(row{table, cols, vals}); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		log.Printf("  ROW COUNT %v\n", it.RowCount)
	}

	return nil
}

func insertIntoDatabaseFrom(cli *spanner.Client, dec interface{ Decode(t any) error }) error {
	mutationsByTable := make(map[string][]*spanner.Mutation)
	for {
		var row row
		if err := dec.Decode(&row); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		if tableName != "" && row.Table != tableName {
			continue
		}

		vals := make([]interface{}, len(row.Vals))
		for i := 0; i < len(row.Vals); i++ {
			if row.Vals[i].Value == nil {
				// zero value is nil
				continue
			}
			vals[i] = row.Vals[i]
		}

		mut := spanner.Insert(row.Table, row.Cols, vals)
		mutationsByTable[row.Table] = append(mutationsByTable[row.Table], mut)
	}

	for _, table := range databaseTableNames(cli) {
		ms, ok := mutationsByTable[table]
		if !ok {
			continue
		}

		log.Printf("INSERT INTO %s VALUES [%v] ...\n", table, len(ms))
		if _, err := cli.Apply(context.TODO(), ms); err != nil {
			return err
		}
	}

	return nil
}

// databaseTableNames returns tables sorted by parent-child order.
func databaseTableNames(cli *spanner.Client) (tableNames []string) {
	revDepTree := make(map[string]string)

	cli.ReadOnlyTransaction().Query(context.TODO(),
		spanner.NewStatement(`select TABLE_NAME, PARENT_TABLE_NAME from information_schema.Tables where TABLE_TYPE = 'BASE TABLE';`),
	).Do(func(r *spanner.Row) error {
		var tableName string
		if err := r.Column(0, &tableName); err != nil {
			return err
		}
		var parentTableName spanner.NullString
		if err := r.Column(1, &parentTableName); err != nil {
			return err
		}
		if parentTableName.Valid {
			revDepTree[tableName] = parentTableName.StringVal
		} else {
			tableNames = append(tableNames, tableName)
		}
		return nil
	})

	for len(revDepTree) > 0 {
		for table, parentTable := range revDepTree {
			var parentInList bool
			for i := 0; i < len(tableNames) && !parentInList; i++ {
				parentInList = tableNames[i] == parentTable
			}
			if parentInList {
				tableNames = append(tableNames, table)
				delete(revDepTree, table)
			}
		}
	}

	return
}
