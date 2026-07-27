package session

import "testing"

func TestNoteFrameLoadStartAndFinish(t *testing.T) {
	sess := &Session{}

	previousCount, currentCount := sess.noteFrameLoadStart("frame-1")
	if previousCount != 0 || currentCount != 1 {
		t.Fatalf("first start = (%d, %d), want (0, 1)", previousCount, currentCount)
	}

	previousCount, currentCount = sess.noteFrameLoadStart("frame-1")
	if previousCount != 1 || currentCount != 2 {
		t.Fatalf("second start = (%d, %d), want (1, 2)", previousCount, currentCount)
	}

	previousCount, currentCount, tracked := sess.noteFrameLoadFinished("frame-1")
	if !tracked || previousCount != 2 || currentCount != 1 {
		t.Fatalf("first finish = tracked:%v counts:(%d, %d), want true and (2, 1)", tracked, previousCount, currentCount)
	}

	snapshot := sess.loadingFrameSnapshot()
	if got := snapshot["frame-1"]; got != 1 {
		t.Fatalf("snapshot count = %d, want 1", got)
	}

	previousCount, currentCount, tracked = sess.noteFrameLoadFinished("frame-1")
	if !tracked || previousCount != 1 || currentCount != 0 {
		t.Fatalf("second finish = tracked:%v counts:(%d, %d), want true and (1, 0)", tracked, previousCount, currentCount)
	}

	if snapshot := sess.loadingFrameSnapshot(); len(snapshot) != 0 {
		t.Fatalf("final snapshot = %v, want empty", snapshot)
	}
}

func TestNoteFrameLoadFinishedWithoutStart(t *testing.T) {
	sess := &Session{}

	previousCount, currentCount, tracked := sess.noteFrameLoadFinished("missing-frame")
	if tracked || previousCount != 0 || currentCount != 0 {
		t.Fatalf("finish without start = tracked:%v counts:(%d, %d), want false and zeros", tracked, previousCount, currentCount)
	}
}

func TestNoteFrameLoadDetachedClearsTrackedFrame(t *testing.T) {
	sess := &Session{}

	sess.noteFrameLoadStart("frame-1")
	sess.noteFrameLoadStart("frame-1")

	previousCount, tracked := sess.noteFrameLoadDetached("frame-1")
	if !tracked || previousCount != 2 {
		t.Fatalf("detach = tracked:%v previousCount:%d, want true and 2", tracked, previousCount)
	}

	if snapshot := sess.loadingFrameSnapshot(); len(snapshot) != 0 {
		t.Fatalf("snapshot after detach = %v, want empty", snapshot)
	}

	previousCount, tracked = sess.noteFrameLoadDetached("frame-1")
	if tracked || previousCount != 0 {
		t.Fatalf("second detach = tracked:%v previousCount:%d, want false and 0", tracked, previousCount)
	}
}
