package discovery

import (
	"context"
	"os"

	"libs.altipla.consulting/env"
	"libs.altipla.consulting/errors"
	"libs.altipla.consulting/rdb"
	"libs.altipla.consulting/secrets"
)

func OpenRavenDB(dbname string, opts ...rdb.OpenOption) (*rdb.Database, error) {
	if env.IsLocal() {
		address := "http://localhost:13000" // development directly in the machine, like tests
		if local := os.Getenv("LOCAL_RAVENDB"); local != "" {
			address = local // development inside containers
		}
		return rdb.Open(address, dbname, opts...)
	}

	credentials, err := secrets.NewValue(context.Background(), "ravendb-client-credentials")
	if err != nil {
		return nil, errors.Trace(err)
	}
	return rdb.OpenSecret(credentials, dbname, opts...)
}
