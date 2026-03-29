import { resolve } from "node:path";
import { cli } from "@jterrazz/test";
import { MockApiServer } from "./mock-api/server.js";

// Build the binary path
const SPWN_BIN = resolve(import.meta.dirname, "../../bin/spwn");

// Mock API server (shared across tests)
export const mockApi = new MockApiServer(9999);

export const spwn = await cli({
  command: SPWN_BIN,
  root: resolve(import.meta.dirname, "../fixtures"),
});
