package session

import "testing"

func TestNoteFrameLoadStartAndFinish(t *testing.T) {
	sess := &Session{}

	alreadyLoading, tracked := sess.noteFrameLoadStart("frame-1")
	if alreadyLoading || !tracked {
		t.Fatalf("first start = alreadyLoading:%v tracked:%v, want false and true", alreadyLoading, tracked)
	}

	if snapshot := sess.loadingFrameSnapshot(); len(snapshot) != 1 || snapshot[0] != "frame-1" {
		t.Fatalf("snapshot after start = %v, want [frame-1]", snapshot)
	}

	if tracked := sess.noteFrameLoadFinished("frame-1"); !tracked {
		t.Fatal("finish after start was not tracked")
	}

	if snapshot := sess.loadingFrameSnapshot(); len(snapshot) != 0 {
		t.Fatalf("final snapshot = %v, want empty", snapshot)
	}
}

func TestNoteFrameLoadDuplicateStartsNeedOneFinish(t *testing.T) {
	sess := &Session{}

	for i := range 3 {
		alreadyLoading, tracked := sess.noteFrameLoadStart("frame-1")
		if !tracked {
			t.Fatalf("start %d was not tracked", i+1)
		}
		if wantAlreadyLoading := i > 0; alreadyLoading != wantAlreadyLoading {
			t.Fatalf("start %d alreadyLoading = %v, want %v", i+1, alreadyLoading, wantAlreadyLoading)
		}
	}

	if snapshot := sess.loadingFrameSnapshot(); len(snapshot) != 1 || snapshot[0] != "frame-1" {
		t.Fatalf("snapshot after duplicate starts = %v, want [frame-1]", snapshot)
	}

	if tracked := sess.noteFrameLoadFinished("frame-1"); !tracked {
		t.Fatal("single finish after duplicate starts was not tracked")
	}
	if snapshot := sess.loadingFrameSnapshot(); len(snapshot) != 0 {
		t.Fatalf("snapshot after finish = %v, want empty", snapshot)
	}
}

func TestNoteFrameLoadSequentialLoads(t *testing.T) {
	sess := &Session{}

	for cycle := range 2 {
		alreadyLoading, tracked := sess.noteFrameLoadStart("frame-1")
		if alreadyLoading || !tracked {
			t.Fatalf("cycle %d start = alreadyLoading:%v tracked:%v, want false and true", cycle+1, alreadyLoading, tracked)
		}
		if tracked := sess.noteFrameLoadFinished("frame-1"); !tracked {
			t.Fatalf("cycle %d finish was not tracked", cycle+1)
		}
	}
}

func TestNoteFrameLoadFinishedWithoutStart(t *testing.T) {
	sess := &Session{}

	if tracked := sess.noteFrameLoadFinished("missing-frame"); tracked {
		t.Fatal("finish without start was tracked")
	}
	if tracked := sess.noteFrameLoadFinished(""); tracked {
		t.Fatal("finish with empty frame id was tracked")
	}
}

func TestNoteFrameLoadDetachedClearsTrackedFrame(t *testing.T) {
	sess := &Session{}

	sess.noteFrameLoadStart("frame-1")
	sess.noteFrameLoadStart("frame-1")

	if tracked := sess.noteFrameLoadDetached("frame-1"); !tracked {
		t.Fatal("detach after duplicate starts was not tracked")
	}
	if snapshot := sess.loadingFrameSnapshot(); len(snapshot) != 0 {
		t.Fatalf("snapshot after detach = %v, want empty", snapshot)
	}

	if tracked := sess.noteFrameLoadDetached("frame-1"); tracked {
		t.Fatal("second detach was tracked")
	}
}
