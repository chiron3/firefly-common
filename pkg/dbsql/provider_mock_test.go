// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dbsql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/hyperledger/firefly-common/mocks/dbmigratemocks"
	"github.com/hyperledger/firefly-common/pkg/config"
)

// testProvider uses the datadog mocking framework
type mockProvider struct {
	Database
	config config.Section

	mockDB *sql.DB
	mdb    sqlmock.Sqlmock
	mmg    *dbmigratemocks.Driver

	fakePSQLInsert          bool
	openError               error
	getMigrationDriverError error
	individualSort          bool
}

func newMockProvider() *mockProvider {
	config.RootConfigReset()
	conf := config.RootSection("unittest.db")
	conf.AddKnownKey("url", "test")
	mp := &mockProvider{
		config: conf,
		mmg:    &dbmigratemocks.Driver{},
	}
	mp.Database.InitConfig(mp, mp.config)
	mp.config.Set(SQLConfMaxConnections, 10)
	mp.mockDB, mp.mdb, _ = sqlmock.New()
	return mp
}

// init is a convenience to init for tests that aren't testing init itself
func (mp *mockProvider) init() (*mockProvider, sqlmock.Sqlmock) {
	_ = mp.Init(context.Background(), mp, mp.config)
	return mp, mp.mdb
}

func (mp *mockProvider) Name() string {
	return "mockdb"
}

func (mp *mockProvider) SequenceColumn() string {
	return "seq"
}

func (mp *mockProvider) MigrationsDir() string {
	return mp.Name()
}

func (psql *mockProvider) Features() SQLFeatures {
	features := DefaultSQLProviderFeatures()
	features.UseILIKE = true
	features.AcquireLock = func(lockName string) string {
		return fmt.Sprintf(`<acquire lock %s>`, lockName)
	}
	return features
}

func (mp *mockProvider) ApplyInsertQueryCustomizations(insert sq.InsertBuilder, requestConflictEmptyResult bool) (sq.InsertBuilder, bool) {
	if mp.fakePSQLInsert {
		return insert.Suffix(" RETURNING seq"), true
	}
	return insert, false
}

func (mp *mockProvider) Open(url string) (*sql.DB, error) {
	return mp.mockDB, mp.openError
}

func (mp *mockProvider) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return mp.mmg, mp.getMigrationDriverError
}
