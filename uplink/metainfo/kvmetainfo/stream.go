// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package kvmetainfo

import (
	"context"
	"errors"

	"storj.io/storj/pkg/encryption"
	"storj.io/storj/pkg/paths"
	"storj.io/storj/pkg/storj"
	"storj.io/storj/uplink/metainfo"
)

var _ storj.ReadOnlyStream = (*readonlyStream)(nil)

type readonlyStream struct {
	db *DB

	info storj.Object
}

func (stream *readonlyStream) Info() storj.Object { return stream.info }

func (stream *readonlyStream) SegmentsAt(ctx context.Context, byteOffset int64, limit int64) (infos []storj.Segment, more bool, err error) {
	defer mon.Task()(&ctx)(&err)

	if stream.info.FixedSegmentSize <= 0 {
		return nil, false, errors.New("not implemented")
	}

	index := byteOffset / stream.info.FixedSegmentSize
	return stream.Segments(ctx, index, limit)
}

func (stream *readonlyStream) segment(ctx context.Context, index int64) (segment storj.Segment, err error) {
	defer mon.Task()(&ctx)(&err)

	segment = storj.Segment{
		Index: index,
	}

	isLastSegment := segment.Index+1 == stream.info.SegmentCount
	if isLastSegment {
		index = -1
	}
	info, limits, err := stream.db.metainfo.DownloadSegment(ctx, metainfo.DownloadSegmentParams{
		StreamID: stream.Info().ID,
		Position: storj.SegmentPosition{
			Index: int32(index),
		},
	})
	if err != nil {
		return segment, err
	}

	segment.Size = stream.info.Size
	segment.EncryptedKeyNonce = info.SegmentEncryption.EncryptedKeyNonce
	segment.EncryptedKey = info.SegmentEncryption.EncryptedKey

	streamKey, err := encryption.DeriveContentKey(stream.info.Bucket.Name, paths.NewUnencrypted(stream.info.Path), stream.db.encStore)
	if err != nil {
		return segment, err
	}

	contentKey, err := encryption.DecryptKey(segment.EncryptedKey, stream.info.EncryptionParameters.CipherSuite, streamKey, &segment.EncryptedKeyNonce)
	if err != nil {
		return segment, err
	}

	nonce := new(storj.Nonce)
	_, err = encryption.Increment(nonce, segment.Index+1)
	if err != nil {
		return segment, err
	}

	if len(info.EncryptedInlineData) != 0 || len(limits) == 0 {
		inline, err := encryption.Decrypt(info.EncryptedInlineData, stream.info.EncryptionParameters.CipherSuite, contentKey, nonce)
		if err != nil {
			return segment, err
		}
		segment.Inline = inline
	}

	return segment, nil
}

func (stream *readonlyStream) Segments(ctx context.Context, index int64, limit int64) (infos []storj.Segment, more bool, err error) {
	defer mon.Task()(&ctx)(&err)

	if index < 0 {
		return nil, false, errors.New("invalid argument")
	}
	if limit <= 0 {
		limit = defaultSegmentLimit
	}
	if index >= stream.info.SegmentCount {
		return nil, false, nil
	}

	infos = make([]storj.Segment, 0, limit)
	for ; index < stream.info.SegmentCount && limit > 0; index++ {
		limit--
		segment, err := stream.segment(ctx, index)
		if err != nil {
			return nil, false, err
		}
		infos = append(infos, segment)
	}

	more = index < stream.info.SegmentCount
	return infos, more, nil
}

type mutableStream struct {
	db   *DB
	info storj.Object
}

func (stream *mutableStream) Info() storj.Object { return stream.info }

func (stream *mutableStream) AddSegments(ctx context.Context, segments ...storj.Segment) (err error) {
	defer mon.Task()(&ctx)(&err)
	return errors.New("not implemented")
}

func (stream *mutableStream) UpdateSegments(ctx context.Context, segments ...storj.Segment) (err error) {
	defer mon.Task()(&ctx)(&err)
	return errors.New("not implemented")
}
