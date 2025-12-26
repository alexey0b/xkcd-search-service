package xkcd_test

import (
	"context"
	"log/slog"
	"search-service/update/adapters/xkcd"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const baseUrl = "https://xkcd.com"

func TestNewClient(t *testing.T) {
	testCases := []struct {
		desc    string
		url     string
		wantErr bool
	}{
		{
			desc: "success - valide client",
			url:  baseUrl,
		},
		{
			desc:    "error - invalid client cause empty url",
			url:     "",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			client, err := xkcd.NewClient(tc.url, 1*time.Minute, slog.Default())
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, client)
			}
		})
	}
}

func TestGet(t *testing.T) {
	testCases := []struct {
		desc    string
		url     string
		id      int64
		wantErr bool
	}{
		{
			desc: "success - valide GET request",
			url:  baseUrl,
			id:   1,
		},
		{
			desc:    "error - not found cause zero id",
			url:     baseUrl,
			id:      0,
			wantErr: true,
		},
		{
			desc:    "error - not found cause negative id",
			url:     baseUrl,
			id:      -1,
			wantErr: true,
		},
		{
			desc:    "error - invalid domain",
			url:     "http://invalid-domain-12345.com",
			id:      1,
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			client, err := xkcd.NewClient(tc.url, 1*time.Minute, slog.Default())
			require.NoError(t, err)
			info, err := client.Get(context.TODO(), tc.id)
			if tc.wantErr {
				require.Error(t, err)
				require.Empty(t, info)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, info)
			}
		})
	}
}

func TestLastID(t *testing.T) {
	testCases := []struct {
		desc    string
		url     string
		wantErr bool
	}{
		{
			desc: "success - valide GET request",
			url:  baseUrl,
		},
		{
			desc:    "error - invalid domain",
			url:     "http://invalid-domain-12345.com",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			client, err := xkcd.NewClient(tc.url, 1*time.Minute, slog.Default())
			require.NoError(t, err)
			lastID, err := client.LastID(context.TODO())
			if tc.wantErr {
				require.Error(t, err)
				require.Zero(t, lastID)
			} else {
				require.NoError(t, err)
				require.Positive(t, lastID)
			}
		})
	}
}
