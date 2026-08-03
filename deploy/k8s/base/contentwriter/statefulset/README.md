# Contentwriter StatefulSet

This base defines contentwriter as a StatefulSet. Each replica receives its own
`ReadWriteOnce` `warcs` claim. The claims are retained when the StatefulSet is
scaled down or deleted and are reused when the same ordinal returns.

Overlays should set an appropriate storage class and capacity. Deleting a PVC
can still delete its backing volume when the storage class uses the `Delete`
reclaim policy, so these claims are not a substitute for the S3 archive.

After deployment:

1. Verify that every `warcs-contentwriter-<ordinal>` claim is bound and each pod
   is ready.
2. Delete one pod and verify that its replacement reattaches the same claim.
3. Confirm that WARC files and fallback files are uploaded and removed after a
   temporary S3 failure.
4. Monitor PVC usage and the storage class's thin-pool capacity, and expand the
   claims before they fill.
