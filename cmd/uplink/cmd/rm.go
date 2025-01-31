// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"storj.io/storj/pkg/process"
	"storj.io/storj/private/fpath"
)

func init() {
	addCmd(&cobra.Command{
		Use:   "rm",
		Short: "Delete an object",
		RunE:  deleteObject,
	}, RootCmd)
}

func deleteObject(cmd *cobra.Command, args []string) error {
	ctx, _ := process.Ctx(cmd)

	if len(args) == 0 {
		return fmt.Errorf("no object specified for deletion")
	}

	dst, err := fpath.New(args[0])
	if err != nil {
		return err
	}

	if dst.IsLocal() {
		return fmt.Errorf("no bucket specified, use format sj://bucket/")
	}

	project, bucket, err := cfg.GetProjectAndBucket(ctx, dst.Bucket())
	if err != nil {
		return convertError(err, dst)
	}
	defer closeProjectAndBucket(project, bucket)

	err = bucket.DeleteObject(ctx, dst.Path())
	if err != nil {
		return convertError(err, dst)
	}

	fmt.Printf("Deleted %s\n", dst)

	return nil
}
