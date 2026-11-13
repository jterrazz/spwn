package contracttest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
)

type Subject struct {
	Jobs contract.Jobs
	UID  int
	// value: a folder the definitions are written under.
	Dir string
}

func spec(s Subject, label string) contract.JobSpec {
	return contract.JobSpec{
		Label: label, Program: []string{"/usr/local/bin/agent", "serve"}, Env: map[string]string{"HOME": s.Dir},
		AtLogin: true, KeepAlive: true, Log: s.Dir + "/logs/" + label + ".log", File: s.Dir + "/" + label + ".plist",
	}
}

func ref(s Subject, label string) contract.Ref {
	return contract.Ref{Domain: contract.Domain{UID: s.UID}, Label: label, File: s.Dir + "/" + label + ".plist"}
}

func listed(t *testing.T, s Subject, label string) (background.Job, bool) {
	t.Helper()
	jobs, err := s.Jobs.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	i := slices.IndexFunc(jobs, func(j background.Job) bool { return j.Label == label })
	if i < 0 {
		return background.Job{}, false
	}
	return jobs[i], true
}

func Jobs(t *testing.T, subject func(t *testing.T) Subject) {
	ctx := context.Background()
	const label = "com.example.agent"

	t.Run("an installed job is listed, and runs only once it is loaded and started", func(t *testing.T) {
		s := subject(t)
		if changed, err := s.Jobs.Install(ctx, spec(s, label)); err != nil || !changed {
			t.Fatalf("Install = %v, %v; want a written definition", changed, err)
		}
		if s.Jobs.Read(ctx, ref(s, label)).Loaded {
			t.Fatal("a job only installed reads loaded")
		}
		if err := s.Jobs.Load(ctx, ref(s, label)); err != nil {
			t.Fatalf("Load: %v", err)
		}
		if err := s.Jobs.Start(ctx, ref(s, label)); err != nil {
			t.Fatalf("Start: %v", err)
		}
		job, ok := listed(t, s, label)
		if !ok || !job.Loaded || !job.Running || job.Program != "/usr/local/bin/agent" {
			t.Fatalf("List gave %+v (found %v); want it loaded and running /usr/local/bin/agent", job, ok)
		}
		if !s.Jobs.Read(ctx, ref(s, label)).Loaded {
			t.Fatal("a started job does not read loaded")
		}
	})

	t.Run("installing the same definition again writes nothing", func(t *testing.T) {
		s := subject(t)
		if _, err := s.Jobs.Install(ctx, spec(s, label)); err != nil {
			t.Fatalf("Install: %v", err)
		}
		if changed, err := s.Jobs.Install(ctx, spec(s, label)); err != nil || changed {
			t.Fatalf("a second Install = %v, %v; want unchanged", changed, err)
		}
		other := spec(s, label)
		other.Program = []string{"/usr/local/bin/agent", "watch"}
		if changed, err := s.Jobs.Install(ctx, other); err != nil || !changed {
			t.Fatalf("Install of a new program = %v, %v; want it written", changed, err)
		}
	})

	t.Run("a stopped job reads not loaded, and stopping it again fails with the command to read it", func(t *testing.T) {
		s := subject(t)
		if _, err := s.Jobs.Install(ctx, spec(s, label)); err != nil {
			t.Fatalf("Install: %v", err)
		}
		if err := s.Jobs.Load(ctx, ref(s, label)); err != nil {
			t.Fatalf("Load: %v", err)
		}
		if err := s.Jobs.Stop(ctx, ref(s, label)); err != nil {
			t.Fatalf("Stop: %v", err)
		}
		if s.Jobs.Read(ctx, ref(s, label)).Loaded {
			t.Fatal("a stopped job reads loaded")
		}
		var failed *contract.Failed
		if err := s.Jobs.Stop(ctx, ref(s, label)); !errors.As(err, &failed) || failed.See == "" || failed.Error() == "" {
			t.Fatalf("a second Stop = %v; want a Failed naming how to read the job", err)
		}
	})

	t.Run("a removed job is gone, and removing it again is no failure", func(t *testing.T) {
		s := subject(t)
		if _, err := s.Jobs.Install(ctx, spec(s, label)); err != nil {
			t.Fatalf("Install: %v", err)
		}
		if err := s.Jobs.Load(ctx, ref(s, label)); err != nil {
			t.Fatalf("Load: %v", err)
		}
		if err := s.Jobs.Remove(ctx, ref(s, label)); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if job, ok := listed(t, s, label); ok {
			t.Fatalf("a removed job is still listed: %+v", job)
		}
		if err := s.Jobs.Remove(ctx, ref(s, label)); err != nil {
			t.Fatalf("a second Remove: %v", err)
		}
	})
}
