package discovery

import (
	"encoding/json"
	"os"

	"libs.altipla.consulting/env"
	"libs.altipla.consulting/errors"
	"libs.altipla.consulting/rdb"
)

func OpenRavenDB(dbname string, opts ...rdb.OpenOption) (*rdb.Database, error) {
	var address string
	if env.IsLocal() {
		address = "http://localhost:13000" // development on the machines
		if local := os.Getenv("LOCAL_RAVENDB"); local != "" {
			address = local // development inside containers
		}
	}

	if !env.IsLocal() {
		var credentials struct {
			Address string
			Key     string
			Cert    string
			CACert  string
		}
		f, err := os.Open("/etc/secrets-v2/ravendb-client-credentials/value")
		if err != nil {
			return nil, errors.Trace(err)
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&credentials); err != nil {
			return nil, errors.Trace(err)
		}
		address = credentials.Address
		opts = append(opts, rdb.WithSecurity(credentials.CACert, credentials.Key, credentials.Cert))
	}

	sess, err := rdb.Open(address, dbname, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return sess, nil
}
