# neo

Fixture soul for the check-legacy-refs e2e. This agent is never
actually spawned — the manifest's bad dep entries make the project
fail `spwn check`, which is the whole point of the fixture.
