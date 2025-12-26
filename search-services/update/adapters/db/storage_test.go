package db_test

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"search-service/update/adapters/db"
	"search-service/update/core"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

func TestAdd(t *testing.T) {
	testCases := []struct {
		desc    string
		comics  []core.Comic
		cleanup func(t *testing.T)
		wantErr bool
	}{
		{
			desc: "success - adds multiple comics",
			comics: []core.Comic{
				{ID: 1, URL: "http://example.com/1", Words: []string{"test", "comic"}},
				{ID: 2, URL: "http://example.com/2", Words: []string{"another"}},
			},
			cleanup: func(t *testing.T) { teardown(t, "comics") },
			wantErr: false,
		},
		{
			desc: "success - adds single comic",
			comics: []core.Comic{
				{ID: 1, URL: "http://example.com/1", Words: []string{"test"}},
			},
			cleanup: func(t *testing.T) { teardown(t, "comics") },
			wantErr: false,
		},
		{
			desc:    "error - zero number of comics",
			comics:  []core.Comic{},
			cleanup: func(t *testing.T) {},
			wantErr: true,
		},
		{
			desc:    "error - nill slice of comics",
			cleanup: func(t *testing.T) {},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			defer tc.cleanup(t)

			err := testDB.Add(context.TODO(), tc.comics...)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			var comicsPg []struct {
				ID    int64          `db:"id"`
				URL   string         `db:"url"`
				Words pq.StringArray `db:"words"`
			}
			err = conn.Select(&comicsPg, "SELECT * FROM comics")

			comics := make([]core.Comic, len(comicsPg))
			for i, comicPg := range comicsPg {
				comics[i] = core.Comic{
					ID:    comicPg.ID,
					URL:   comicPg.URL,
					Words: comicPg.Words,
				}
			}
			require.NoError(t, err)
			require.ElementsMatch(t, comics, tc.comics)
		})
	}
}

func TestStats(t *testing.T) {
	testCases := []struct {
		desc          string
		prepare       func(t *testing.T)
		cleanup       func(t *testing.T)
		expectedStats core.DBStats
		wantErr       bool
	}{
		{
			desc: "success - returns correct stats",
			prepare: func(t *testing.T) {
				_, err := conn.Exec(`
					UPDATE comics_stats 
					SET comics_fetched = 2, words_total = 4, words_unique = 3
				`)
				require.NoError(t, err)
			},
			cleanup: func(t *testing.T) { teardown(t, "comics_stats") },
			expectedStats: core.DBStats{
				WordsTotal:    4,
				WordsUnique:   3,
				ComicsFetched: 2,
			},
			wantErr: false,
		},
		{
			desc:    "success - empty table returns zeros",
			prepare: func(t *testing.T) {},
			cleanup: func(t *testing.T) {},
			expectedStats: core.DBStats{
				WordsTotal:    0,
				WordsUnique:   0,
				ComicsFetched: 0,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.prepare(t)
			defer tc.cleanup(t)

			stats, err := testDB.Stats(context.TODO())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedStats, stats)
			}
		})
	}
}

func TestIDs(t *testing.T) {
	testCases := []struct {
		desc        string
		prepare     func(t *testing.T)
		cleanup     func(t *testing.T)
		expectedIDs []int64
		wantErr     bool
	}{
		{
			desc: "success - returns all IDs",
			prepare: func(t *testing.T) {
				_, err := conn.Exec(`
					INSERT INTO comics (id, url, words) VALUES 
					(1, 'http://example.com/1', ARRAY['test']),
					(2, 'http://example.com/2', ARRAY['another']),
					(3, 'http://example.com/3', ARRAY['third'])
				`)
				require.NoError(t, err)
			},
			cleanup:     func(t *testing.T) { teardown(t, "comics") },
			expectedIDs: []int64{1, 2, 3},
			wantErr:     false,
		},
		{
			desc:        "success - empty table returns empty",
			prepare:     func(t *testing.T) {},
			cleanup:     func(t *testing.T) {},
			expectedIDs: []int64{},
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.prepare(t)
			defer tc.cleanup(t)

			ids, err := testDB.IDs(context.TODO())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tc.expectedIDs, ids)
			}
		})
	}
}

func TestDrop(t *testing.T) {
	testCases := []struct {
		desc    string
		prepare func(t *testing.T)
		cleanup func(t *testing.T)
		wantErr bool
	}{
		{
			desc: "success - drops all comics",
			prepare: func(t *testing.T) {
				_, err := conn.Exec(`
					INSERT INTO comics (id, url, words) VALUES 
					(1, 'http://example.com/1', ARRAY['test']),
					(2, 'http://example.com/2', ARRAY['another'])
				`)
				require.NoError(t, err)
			},
			cleanup: func(t *testing.T) { teardown(t, "comics") },
			wantErr: false,
		},
		{
			desc:    "success - drop on empty table",
			prepare: func(t *testing.T) {},
			cleanup: func(t *testing.T) {},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.prepare(t)
			defer tc.cleanup(t)

			err := testDB.Drop(context.TODO())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var count int
				err := conn.Get(&count, "SELECT COUNT(*) FROM comics")
				require.NoError(t, err)
				require.Equal(t, 0, count)
			}
		})
	}
}

func teardown(t *testing.T, table string) {
	switch table {
	case "comics":
		_, err := conn.Exec("TRUNCATE comics")
		require.NoError(t, err)
	case "comics_stats":
		_, err := conn.Exec("UPDATE comics_stats SET comics_fetched = 0, words_total = 0, words_unique = 0")
		require.NoError(t, err)
	}
}
