package session

import (
	"testing"

	"github.com/gorilla/websocket"
)

type connectionBucketItem struct {
	id   string
	conn *websocket.Conn
}

func connForBucketItem(item connectionBucketItem) *websocket.Conn {
	return item.conn
}

func TestAppendIfConnMissingAddsAndDeduplicates(t *testing.T) {
	conn := &websocket.Conn{}
	items := []connectionBucketItem{}

	items = appendIfConnMissing(items, connectionBucketItem{id: "first", conn: conn}, connForBucketItem)
	if len(items) != 1 {
		t.Fatalf("expected one item after initial append, got %d", len(items))
	}

	items = appendIfConnMissing(items, connectionBucketItem{id: "duplicate", conn: conn}, connForBucketItem)
	if len(items) != 1 {
		t.Fatalf("expected duplicate conn append to be ignored, got %d", len(items))
	}
}

func TestAppendIfConnMissingSkipsNilConn(t *testing.T) {
	items := appendIfConnMissing([]connectionBucketItem{}, connectionBucketItem{id: "nil", conn: nil}, connForBucketItem)
	if len(items) != 0 {
		t.Fatalf("expected nil conn candidate to be ignored, got %d items", len(items))
	}
}

func TestHasConnAndHasConnInBuckets(t *testing.T) {
	connA := &websocket.Conn{}
	connB := &websocket.Conn{}

	items := []connectionBucketItem{
		{id: "a", conn: connA},
		{id: "b", conn: connB},
	}
	if !hasConn(items, connA, connForBucketItem) {
		t.Fatal("expected hasConn to locate connA")
	}
	if hasConn(items, &websocket.Conn{}, connForBucketItem) {
		t.Fatal("expected hasConn to return false for unknown conn")
	}

	buckets := map[string][]connectionBucketItem{
		"token-a": items,
	}
	if !hasConnInBuckets(buckets, connB, connForBucketItem) {
		t.Fatal("expected hasConnInBuckets to locate connB")
	}
}

func TestRemoveConnAndRemoveConnFromBuckets(t *testing.T) {
	connA := &websocket.Conn{}
	connB := &websocket.Conn{}
	items := []connectionBucketItem{
		{id: "a", conn: connA},
		{id: "b", conn: connB},
	}

	filtered, removed := removeConn(items, connA, connForBucketItem)
	if !removed {
		t.Fatal("expected removeConn to remove connA")
	}
	if len(filtered) != 1 || filtered[0].id != "b" {
		t.Fatalf("unexpected filtered slice after remove: %+v", filtered)
	}

	buckets := map[string][]connectionBucketItem{
		"token-a": {
			{id: "a", conn: connA},
		},
		"token-b": {
			{id: "b", conn: connB},
		},
	}
	if !removeConnFromBuckets(buckets, connA, connForBucketItem) {
		t.Fatal("expected removeConnFromBuckets to remove connA")
	}
	if _, exists := buckets["token-a"]; exists {
		t.Fatal("expected empty token bucket to be deleted")
	}
	if len(buckets["token-b"]) != 1 {
		t.Fatalf("expected unrelated bucket to remain unchanged, got %+v", buckets["token-b"])
	}
}

func TestSnapshotHelpersCloneCollectionShapes(t *testing.T) {
	conn := &websocket.Conn{}
	sourceSlice := []connectionBucketItem{{id: "a", conn: conn}}
	snapshot := snapshotSlice(sourceSlice, func(item connectionBucketItem) connectionBucketItem { return item })
	if len(snapshot) != 1 || snapshot[0].id != "a" {
		t.Fatalf("unexpected snapshot slice contents: %+v", snapshot)
	}
	sourceSlice[0].id = "mutated"
	if snapshot[0].id != "a" {
		t.Fatalf("expected snapshot slice to remain stable after source mutation, got %+v", snapshot)
	}

	sourceBuckets := map[string][]connectionBucketItem{
		"token": {{id: "bucket-item", conn: conn}},
	}
	bucketSnapshot := snapshotBuckets(sourceBuckets, func(item connectionBucketItem) connectionBucketItem { return item })
	if len(bucketSnapshot["token"]) != 1 || bucketSnapshot["token"][0].id != "bucket-item" {
		t.Fatalf("unexpected snapshot bucket contents: %+v", bucketSnapshot)
	}
	sourceBuckets["token"][0].id = "changed"
	if bucketSnapshot["token"][0].id != "bucket-item" {
		t.Fatalf("expected snapshot bucket to remain stable after source mutation, got %+v", bucketSnapshot)
	}
}

func TestClonePtrConnReturnsSamePointer(t *testing.T) {
	conn := &websocket.Conn{}
	if clone := clonePtrConn(conn); clone != conn {
		t.Fatal("expected clonePtrConn to preserve pointer identity")
	}
}
