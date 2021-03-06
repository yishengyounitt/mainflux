// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mainflux/mainflux/pkg/errors"
	uuidProvider "github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
)

func TestChannelsSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	channelRepo := postgres.NewChannelRepository(dbMiddleware)

	email := "channel-save@example.com"

	chs := []things.Channel{}
	for i := 1; i <= 5; i++ {
		id, err := uuidProvider.New().ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    id,
			Owner: email,
		}
		chs = append(chs, ch)
	}
	id := chs[0].ID

	cases := []struct {
		desc     string
		channels []things.Channel
		err      error
	}{
		{
			desc:     "create new channels",
			channels: chs,
			err:      nil,
		},
		{
			desc:     "create channels that already exist",
			channels: chs,
			err:      things.ErrConflict,
		},
		{
			desc: "create channel with invalid ID",
			channels: []things.Channel{
				{ID: "invalid", Owner: email},
			},
			err: things.ErrMalformedEntity,
		},
		{
			desc: "create channel with invalid name",
			channels: []things.Channel{
				{ID: id, Owner: email, Name: invalidName},
			},
			err: things.ErrMalformedEntity,
		},
		{
			desc: "create channel with invalid name",
			channels: []things.Channel{
				{ID: id, Owner: email, Name: invalidName},
			},
			err: things.ErrMalformedEntity,
		},
	}

	for _, cc := range cases {
		_, err := channelRepo.Save(context.Background(), cc.channels...)
		assert.True(t, errors.Contains(err, cc.err), fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestChannelUpdate(t *testing.T) {
	email := "channel-update@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	id, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Channel{
		ID:    id,
		Owner: email,
	}

	chs, err := chanRepo.Save(context.Background(), ch)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch.ID = chs[0].ID

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		channel things.Channel
		err     error
	}{
		{
			desc:    "update existing channel",
			channel: ch,
			err:     nil,
		},
		{
			desc: "update non-existing channel with existing user",
			channel: things.Channel{
				ID:    nonexistentChanID,
				Owner: email,
			},
			err: things.ErrNotFound,
		},
		{
			desc: "update existing channel ID with non-existing user",
			channel: things.Channel{
				ID:    ch.ID,
				Owner: wrongValue,
			},
			err: things.ErrNotFound,
		},
		{
			desc: "update non-existing channel with non-existing user",
			channel: things.Channel{
				ID:    nonexistentChanID,
				Owner: wrongValue,
			},
			err: things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Update(context.Background(), tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleChannelRetrieval(t *testing.T) {
	email := "channel-single-retrieval@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}
	ths, _ := thingRepo.Save(context.Background(), th)
	th.ID = ths[0].ID

	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Channel{
		ID:    chid,
		Owner: email,
	}
	chs, _ := chanRepo.Save(context.Background(), ch)
	ch.ID = chs[0].ID
	chanRepo.Connect(context.Background(), email, []string{ch.ID}, []string{th.ID})

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		owner string
		ID    string
		err   error
	}{
		"retrieve channel with existing user": {
			owner: ch.Owner,
			ID:    ch.ID,
			err:   nil,
		},
		"retrieve channel with existing user, non-existing channel": {
			owner: ch.Owner,
			ID:    nonexistentChanID,
			err:   things.ErrNotFound,
		},
		"retrieve channel with non-existing owner": {
			owner: wrongValue,
			ID:    ch.ID,
			err:   things.ErrNotFound,
		},
		"retrieve channel with malformed ID": {
			owner: ch.Owner,
			ID:    wrongValue,
			err:   things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := chanRepo.RetrieveByID(context.Background(), tc.owner, tc.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiChannelRetrieval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	email := "channel-multi-retrieval@example.com"
	name := "channel_name"
	metadata := things.Metadata{
		"field": "value",
	}
	wrongMeta := things.Metadata{
		"wrong": "wrong",
	}

	offset := uint64(1)
	nameNum := uint64(3)
	metaNum := uint64(3)
	nameMetaNum := uint64(2)

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		chid, err := uuidProvider.New().ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    chid,
			Owner: email,
		}

		// Create Channels with name.
		if i < nameNum {
			ch.Name = name
		}
		// Create Channels with metadata.
		if i >= nameNum && i < nameNum+metaNum {
			ch.Metadata = metadata
		}
		// Create Channels with name and metadata.
		if i >= n-nameMetaNum {
			ch.Metadata = metadata
			ch.Name = name
		}

		chanRepo.Save(context.Background(), ch)
	}

	cases := map[string]struct {
		owner    string
		offset   uint64
		limit    uint64
		name     string
		size     uint64
		total    uint64
		metadata things.Metadata
	}{
		"retrieve all channels with existing owner": {
			owner:  email,
			offset: 0,
			limit:  n,
			size:   n,
			total:  n,
		},
		"retrieve subset of channels with existing owner": {
			owner:  email,
			offset: n / 2,
			limit:  n,
			size:   n / 2,
			total:  n,
		},
		"retrieve channels with non-existing owner": {
			owner:  wrongValue,
			offset: n / 2,
			limit:  n,
			size:   0,
			total:  0,
		},
		"retrieve channels with existing name": {
			owner:  email,
			offset: offset,
			limit:  n,
			name:   name,
			size:   nameNum + nameMetaNum - offset,
			total:  nameNum + nameMetaNum,
		},
		"retrieve all channels with non-existing name": {
			owner:  email,
			offset: 0,
			limit:  n,
			name:   "wrong",
			size:   0,
			total:  0,
		},
		"retrieve all channels with existing metadata": {
			owner:    email,
			offset:   0,
			limit:    n,
			size:     metaNum + nameMetaNum,
			total:    metaNum + nameMetaNum,
			metadata: metadata,
		},
		"retrieve all channels with non-existing metadata": {
			owner:    email,
			offset:   0,
			limit:    n,
			total:    0,
			metadata: wrongMeta,
		},
		"retrieve all channels with existing name and metadata": {
			owner:    email,
			offset:   0,
			limit:    n,
			size:     nameMetaNum,
			total:    nameMetaNum,
			name:     name,
			metadata: metadata,
		},
	}

	for desc, tc := range cases {
		page, err := chanRepo.RetrieveAll(context.Background(), tc.owner, tc.offset, tc.limit, tc.name, tc.metadata)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveByThing(t *testing.T) {
	email := "channel-multi-retrieval-by-thing@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ths, err := thingRepo.Save(context.Background(), things.Thing{
		ID:    thid,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thid = ths[0].ID

	n := uint64(10)
	chsDisconNum := uint64(1)

	for i := uint64(0); i < n; i++ {
		chid, err := uuidProvider.New().ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		ch := things.Channel{
			ID:    chid,
			Owner: email,
		}
		schs, err := chanRepo.Save(context.Background(), ch)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		cid := schs[0].ID

		// Don't connect last Channel
		if i == n-chsDisconNum {
			break
		}

		err = chanRepo.Connect(context.Background(), email, []string{cid}, []string{thid})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	nonexistentThingID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		owner     string
		thing     string
		offset    uint64
		limit     uint64
		connected bool
		size      uint64
		err       error
	}{
		"retrieve all channels by thing with existing owner": {
			owner:     email,
			thing:     thid,
			offset:    0,
			limit:     n,
			connected: true,
			size:      n - chsDisconNum,
		},
		"retrieve subset of channels by thing with existing owner": {
			owner:     email,
			thing:     thid,
			offset:    n / 2,
			limit:     n,
			connected: true,
			size:      (n / 2) - chsDisconNum,
		},
		"retrieve channels by thing with non-existing owner": {
			owner:     wrongValue,
			thing:     thid,
			offset:    n / 2,
			limit:     n,
			connected: true,
			size:      0,
		},
		"retrieve channels by non-existent thing": {
			owner:     email,
			thing:     nonexistentThingID,
			offset:    0,
			limit:     n,
			connected: true,
			size:      0,
		},
		"retrieve channels with malformed UUID": {
			owner:     email,
			thing:     wrongValue,
			offset:    0,
			limit:     n,
			connected: true,
			size:      0,
			err:       things.ErrNotFound,
		},
		"retrieve all non connected channels by thing with existing owner": {
			owner:     email,
			thing:     thid,
			offset:    0,
			limit:     n,
			connected: false,
			size:      chsDisconNum,
		},
	}

	for desc, tc := range cases {
		page, err := chanRepo.RetrieveByThing(context.Background(), tc.owner, tc.thing, tc.offset, tc.limit, tc.connected)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestChannelRemoval(t *testing.T) {
	email := "channel-removal@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chid = chs[0].ID

	// show that the removal works the same for both existing and non-existing
	// (removed) channel
	for i := 0; i < 2; i++ {
		err := chanRepo.Remove(context.Background(), email, chid)
		require.Nil(t, err, fmt.Sprintf("#%d: failed to remove channel due to: %s", i, err))

		_, err = chanRepo.RetrieveByID(context.Background(), email, chid)
		assert.True(t, errors.Contains(err, things.ErrNotFound), fmt.Sprintf("#%d: expected %s got %s", i, things.ErrNotFound, err))
	}
}

func TestConnect(t *testing.T) {
	email := "channel-connect@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := things.Thing{
		ID:       thid,
		Owner:    email,
		Key:      thkey,
		Metadata: things.Metadata{},
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thid = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chid = chs[0].ID

	nonexistentThingID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		owner string
		chid  string
		thid  string
		err   error
	}{
		{
			desc:  "connect existing user, channel and thing",
			owner: email,
			chid:  chid,
			thid:  thid,
			err:   nil,
		},
		{
			desc:  "connect connected channel and thing",
			owner: email,
			chid:  chid,
			thid:  thid,
			err:   things.ErrConflict,
		},
		{
			desc:  "connect with non-existing user",
			owner: wrongValue,
			chid:  chid,
			thid:  thid,
			err:   things.ErrNotFound,
		},
		{
			desc:  "connect non-existing channel",
			owner: email,
			chid:  nonexistentChanID,
			thid:  thid,
			err:   things.ErrNotFound,
		},
		{
			desc:  "connect non-existing thing",
			owner: email,
			chid:  chid,
			thid:  nonexistentThingID,
			err:   things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Connect(context.Background(), tc.owner, []string{tc.chid}, []string{tc.thid})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	email := "channel-disconnect@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:       thid,
		Owner:    email,
		Key:      thkey,
		Metadata: map[string]interface{}{},
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thid = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chid = chs[0].ID
	chanRepo.Connect(context.Background(), email, []string{chid}, []string{thid})

	nonexistentThingID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		owner string
		chid  string
		thid  string
		err   error
	}{
		{
			desc:  "disconnect connected thing",
			owner: email,
			chid:  chid,
			thid:  thid,
			err:   nil,
		},
		{
			desc:  "disconnect non-connected thing",
			owner: email,
			chid:  chid,
			thid:  thid,
			err:   things.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing user",
			owner: wrongValue,
			chid:  chid,
			thid:  thid,
			err:   things.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing channel",
			owner: email,
			chid:  nonexistentChanID,
			thid:  thid,
			err:   things.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing thing",
			owner: email,
			chid:  chid,
			thid:  nonexistentThingID,
			err:   things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Disconnect(context.Background(), tc.owner, tc.chid, tc.thid)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestHasThing(t *testing.T) {
	email := "channel-access-check@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thid = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chid = chs[0].ID
	chanRepo.Connect(context.Background(), email, []string{chid}, []string{thid})

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		chid      string
		key       string
		hasAccess bool
	}{
		"access check for thing that has access": {
			chid:      chid,
			key:       th.Key,
			hasAccess: true,
		},
		"access check for thing without access": {
			chid:      chid,
			key:       wrongValue,
			hasAccess: false,
		},
		"access check for non-existing channel": {
			chid:      nonexistentChanID,
			key:       th.Key,
			hasAccess: false,
		},
	}

	for desc, tc := range cases {
		_, err := chanRepo.HasThing(context.Background(), tc.chid, tc.key)
		hasAccess := err == nil
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}

func TestHasThingByID(t *testing.T) {
	email := "channel-access-check@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thid = ths[0].ID

	disconnectedThID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	disconnectedThKey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	disconnectedThing := things.Thing{
		ID:    disconnectedThID,
		Owner: email,
		Key:   disconnectedThKey,
	}
	ths, err = thingRepo.Save(context.Background(), disconnectedThing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	disconnectedThingID := ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chid = chs[0].ID
	chanRepo.Connect(context.Background(), email, []string{chid}, []string{thid})

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		chid      string
		thid      string
		hasAccess bool
	}{
		"access check for thing that has access": {
			chid:      chid,
			thid:      thid,
			hasAccess: true,
		},
		"access check for thing without access": {
			chid:      chid,
			thid:      disconnectedThingID,
			hasAccess: false,
		},
		"access check for non-existing channel": {
			chid:      nonexistentChanID,
			thid:      thid,
			hasAccess: false,
		},
		"access check for non-existing thing": {
			chid:      chid,
			thid:      wrongValue,
			hasAccess: false,
		},
	}

	for desc, tc := range cases {
		err := chanRepo.HasThingByID(context.Background(), tc.chid, tc.thid)
		hasAccess := err == nil
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}
