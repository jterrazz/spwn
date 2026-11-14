# @jterrazz/station-api

The wire contract of `station.api/v2`, machine-readable: a JSON Schema per call and the TypeScript a client types its calls with, all written by `make generate` from station's Go types.

| File                   | Holds                                                                                                                             |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `schemas/calls/*.json` | One schema per call or watch of `apps/engine/remote/api/calls.json`: its `in`, and the `out` of each answer                       |
| `schemas/frame.json`   | The frame every line is: a request (`hello`, `call`, `watch`) or an answer (`hello`, `progress`, `out`, `error`, `done`)          |
| `schemas/event.json`   | The `out` of the watches `page` and `engine.events`: their opening, then `page`, `sample`, `tick`, `notice`                       |
| `index.d.ts`           | Every schema as a TypeScript type, and `Calls`, the map from a call's name to its input and output                                |
| `words.json`           | The words a client draws for each value of a closed enum (word, detail, heading, nouns after a count, tone), typed `StationWords` |
| `words.d.ts`           | The type of `words.json`, `StationWords`, which the `./words.json` export resolves to                                             |

Nothing here is written by hand but this page and `package.json`: the generator is `apps/engine/cmd/generate/` (`api.go`, `schema.go`, `ts.go`, and `words.go`, the words table), and its drift test fails `make test` when a file and the Go types differ.

## How the schemas read the Go types

- **An answer is what `call.Encode` writes.** Its encoder is `encoding/json`, but for a struct that holds an `error`, which it walks itself: there a key is left out only under `omitempty` and only when its value is zero, so an `omitzero` time is written as the zero time, an empty slice as `[]`, and `,string` is not read. Elsewhere `encoding/json`'s rules hold: a key is its json tag or its field's name, one under `omitzero`, or under `omitempty` with a type it can find empty, is optional, and every other key is required; a nil slice, map or pointer that is not left out is `null`.
- **An enum's empty value is written.** A string enum whose key is never left out admits `""`, the zero Go sends when no value was set, and so does its TypeScript union; a VM's `Lifetime` and an Origin's `Layer` do not, since the engine always sets them (`neverZero` in `schema.go`).
- **An input is what a client should send.** Every key is optional, since an absent one reads as its zero, none is `null`, and no key outside the schema is named (`additionalProperties: false`, exact names). The server is laxer, as `encoding/json` reads: it ignores an unknown key, reads `null` as the zero, and matches a key's name regardless of case. A client holds to the schema; the server's tolerance is no contract.
- **A string type with constants is an enum** when the code gives it only those constants; a type converted from any other string stays a plain string.
- **Untyped is said as such.** A value Go keeps as `json.RawMessage` (a frame's `in` and `out`, an install's body, a VM's accounts) is `unknown` here; a frame's `in` and `out` are the call's own schema's.

The answers are proved against the schemas, beside the drift test: the recorded streams and page fixtures in `apps/engine/cmd/generate/`, and every read the Registry serves without a confirmation, sent over every world of the spec build, in `specs/architecture/answers_test.go`.

The protocol itself (the transport, the frames, the errors, what each engine call does) is [the API reference](../../docs/reference/api.md).
