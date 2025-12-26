package db_test

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"search-service/search/adapters/db"
	"search-service/search/core"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	psqlC  testcontainers.Container
	conn   *sqlx.DB
	testDB *db.DB
)

func TestMain(m *testing.M) {
	buildContext, err := filepath.Abs("./testdata")
	if err != nil {
		log.Fatalf("failed to resolve absolute path: %v\n", err)
	}

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context: buildContext,
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	}

	psqlC, err = testcontainers.GenericContainer(context.TODO(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatal(err)
	}

	host, err := psqlC.Host(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	mappedPort, err := psqlC.MappedPort(context.TODO(), "5432")
	if err != nil {
		log.Fatal(err)
	}

	psqlURL := fmt.Sprintf(
		"postgres://user:password@%s:%s/test_db?sslmode=disable",
		host,
		mappedPort.Port(),
	)

	conn, err = sqlx.Connect("pgx", psqlURL)
	if err != nil {
		log.Fatalln("failed to connect to database:", err)
	}

	testDB, err = db.New(slog.Default(), psqlURL)
	if err != nil {
		log.Fatalln("failed to connect to database:", err)
	}

	code := m.Run()

	err = testcontainers.TerminateContainer(psqlC)
	if err != nil {
		log.Fatalln("failed to terminate container:", err)
	}

	testDB.Close()
	os.Exit(code)
}

func TestGetComicsByIds(t *testing.T) {
	testCases := []struct {
		desc           string
		requestedIds   []int64
		prepare        func(t *testing.T)
		cleanup        func(t *testing.T)
		expectedComics []core.Comic
		wantErr        bool
	}{
		{
			desc:         "success - returns multiple comics",
			requestedIds: []int64{1, 2, 3},
			prepare: func(t *testing.T) {
				_, err := conn.Exec(`
					INSERT INTO comics (id, url, words) VALUES 
					(1, 'http://example.com/1', ARRAY['test', 'comic']),
					(2, 'http://example.com/2', ARRAY['another', 'test']),
					(3, 'http://example.com/3', ARRAY['third', 'comic'])
				`)
				require.NoError(t, err)
			},
			cleanup: func(t *testing.T) { teardown(t, "comics") },
			expectedComics: []core.Comic{
				{ID: 1, URL: "http://example.com/1"},
				{ID: 2, URL: "http://example.com/2"},
				{ID: 3, URL: "http://example.com/3"},
			},
			wantErr: false,
		},
		{
			desc:         "success - returns single comic",
			requestedIds: []int64{1},
			prepare: func(t *testing.T) {
				_, err := conn.Exec(`
					INSERT INTO comics (id, url, words) VALUES 
					(1, 'http://example.com/1', ARRAY['test'])
				`)
				require.NoError(t, err)
			},
			cleanup:        func(t *testing.T) { teardown(t, "comics") },
			expectedComics: []core.Comic{{ID: 1, URL: "http://example.com/1"}},
			wantErr:        false,
		},
		{
			desc: "success - empty ids returns empty result",
			prepare: func(t *testing.T) {
				_, err := conn.Exec(`
					INSERT INTO comics (id, url, words) VALUES 
					(1, 'http://example.com/1', ARRAY['test'])
				`)
				require.NoError(t, err)
			},
			cleanup:        func(t *testing.T) { teardown(t, "comics") },
			expectedComics: []core.Comic{},
			wantErr:        false,
		},
		{
			desc:         "success - non-existent ids returns empty",
			requestedIds: []int64{3},
			prepare: func(t *testing.T) {
				_, err := conn.Exec(`
					INSERT INTO comics (id, url, words) VALUES 
					(1, 'http://example.com/1', ARRAY['test'])
				`)
				require.NoError(t, err)
			},
			cleanup:        func(t *testing.T) { teardown(t, "comics") },
			expectedComics: []core.Comic{},
			wantErr:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.prepare(t)
			defer tc.cleanup(t)

			comics, err := testDB.GetComicsByIds(context.TODO(), tc.requestedIds)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tc.expectedComics, comics)
			}
		})
	}
}

func TestGetAllComicsInfo(t *testing.T) {
	testCases := []struct {
		desc               string
		prepare            func(t *testing.T)
		cleanup            func(t *testing.T)
		expectedComicsInfo []core.ComicInfo
		wantErr            bool
	}{
		{
			desc: "success - returns all comics with words",
			prepare: func(t *testing.T) {
				_, err := conn.Exec(`
					INSERT INTO comics (id, url, words) VALUES 
					(1, 'http://example.com/1', ARRAY['test', 'comic']),
					(2, 'http://example.com/2', ARRAY['another'])
				`)
				require.NoError(t, err)
			},
			cleanup: func(t *testing.T) { teardown(t, "comics") },
			expectedComicsInfo: []core.ComicInfo{
				{Comic: core.Comic{ID: 1, URL: "http://example.com/1"}, Words: []string{"test", "comic"}},
				{Comic: core.Comic{ID: 2, URL: "http://example.com/2"}, Words: []string{"another"}},
			},
			wantErr: false,
		},
		{
			desc:               "success - empty table returns empty result",
			prepare:            func(t *testing.T) {},
			cleanup:            func(t *testing.T) {},
			expectedComicsInfo: []core.ComicInfo{},
			wantErr:            false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.prepare(t)
			defer tc.cleanup(t)

			comicsInfo, err := testDB.GetAllComicsInfo(context.TODO())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tc.expectedComicsInfo, comicsInfo)
			}
		})
	}
}

func teardown(t *testing.T, table string) {
	_, err := conn.Exec(fmt.Sprintf("TRUNCATE %s", table))
	require.NoError(t, err)
}
