package etcd

import (
	"context"
	"testing"
	"time"

	"github.com/sensu/sensu-go/backend/store"
	"github.com/sensu/sensu-go/types"
	"github.com/stretchr/testify/assert"
)

func TestSilencedStorage(t *testing.T) {
	testWithEtcd(t, func(store store.Store) {
		silenced := types.FixtureSilenced("*:checkname")
		silenced.Organization = "default"
		silenced.Environment = "default"
		silenced.Subscription = "subscription"
		silenced.ID = silenced.Subscription + ":" + silenced.Check
		ctx := context.WithValue(context.Background(), types.OrganizationKey, silenced.Organization)
		ctx = context.WithValue(ctx, types.EnvironmentKey, silenced.Environment)

		// We should receive an empty slice if no results were found
		silencedEntries, err := store.GetSilencedEntries(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, silencedEntries)

		err = store.UpdateSilencedEntry(ctx, silenced)
		if err != nil {
			t.Fatalf("failed to update entry due to error: %s", err)
		}

		// Get all silenced entries
		entries, err := store.GetSilencedEntries(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, entries)
		assert.Equal(t, 1, len(entries))

		// Get silenced entry by id
		entry, err := store.GetSilencedEntryByID(ctx, silenced.ID)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, silenced.Check, entry.Check)

		// Get silenced entry by subscription
		entries, err = store.GetSilencedEntriesBySubscription(ctx, silenced.Subscription)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, 1, len(entries))

		// Get silenced entry by check
		entries, err = store.GetSilencedEntriesByCheckName(ctx, silenced.Check)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, 1, len(entries))

		// Update silenced entry to "wildcard"
		silenced.Check = "*"
		silenced.ID = silenced.Subscription + ":" + silenced.Check
		err = store.UpdateSilencedEntry(ctx, silenced)
		assert.NoError(t, err)

		// Get silenced entry by id with "wildcard"
		entry, err = store.GetSilencedEntryByID(ctx, silenced.ID)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, "subscription:*", entry.ID)
		// Entries without expirations should return -1
		assert.Equal(t, int64(-1), entry.Expire)

		// Delete silenced entry by id
		err = store.DeleteSilencedEntryByID(ctx, silenced.ID)
		assert.NoError(t, err)

		// Update a silenced entry's expire time
		silenced.Expire = 2
		err = store.UpdateSilencedEntry(ctx, silenced)
		assert.NoError(t, err)

		// Wait for the entry to expire
		time.Sleep(3 * time.Second)

		// Check that the entry is deleted
		entry, err = store.GetSilencedEntryByID(ctx, silenced.ID)
		assert.NoError(t, err)
		assert.Nil(t, entry)

		// Updating a silenced entry in a nonexistent org and env should not work
		silenced.Organization = "missing"
		silenced.Environment = "missing"
		err = store.UpdateSilencedEntry(ctx, silenced)
		assert.Error(t, err)

	})
}

func TestSilencedStorageWithExpire(t *testing.T) {
	testWithEtcd(t, func(store store.Store) {
		silenced := types.FixtureSilenced("subscription:checkname")
		silenced.Organization = "default"
		silenced.Environment = "default"
		silenced.Expire = 15
		ctx := context.WithValue(context.Background(), types.OrganizationKey, silenced.Organization)
		ctx = context.WithValue(ctx, types.EnvironmentKey, silenced.Environment)

		err := store.UpdateSilencedEntry(ctx, silenced)
		if err != nil {
			t.Fatalf("failed to update entry due to error: %s", err)
		}

		// Get silenced entry and check that expire time is not zero
		entry, err := store.GetSilencedEntryByID(ctx, silenced.ID)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.NotZero(t, entry.Expire)
	})
}

func TestSilencedStorageWithBegin(t *testing.T) {
	testWithEtcd(t, func(store store.Store) {
		silenced := types.FixtureSilenced("subscription:checkname")
		silenced.Organization = "default"
		silenced.Environment = "default"
		// set a begin time in the future
		silenced.Begin = time.Date(1970, 01, 01, 01, 00, 00, 00, time.UTC).Unix()
		// current time is before the start time
		currentTime := time.Date(1970, 01, 01, 00, 00, 00, 00, time.UTC).Unix()
		ctx := context.WithValue(context.Background(), types.OrganizationKey, silenced.Organization)
		ctx = context.WithValue(ctx, types.EnvironmentKey, silenced.Environment)

		err := store.UpdateSilencedEntry(ctx, silenced)
		if err != nil {
			t.Fatalf("failed to update entry due to error: %s", err)
		}

		// Get silenced entry and check that it is not yet ready to start
		// silencing
		entry, err := store.GetSilencedEntryByID(ctx, silenced.ID)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.False(t, entry.StartSilence(currentTime))
	})
}
