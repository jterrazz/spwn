import Foundation

// invariant: whether the engine is installed is the engine's own answer, read
// through the CLI (`station engine --json`, whose `engine.state` is
// `not-installed` when no job manager holds the engine's job); the menu bar
// names no job file, so it follows the engine into whichever mode holds it.
public enum EngineInstall {
    public static func read(executable: String, launcher: Launcher = ProcessLauncher()) async -> Bool {
        let launched = await launcher.run(executable, ["engine", "--json"], environment: [:])
        guard launched.code == 0,
              let json = try? JSONSerialization.jsonObject(with: Data(launched.stdout.utf8)) as? [String: Any],
              let engine = json["engine"] as? [String: Any],
              let state = engine["state"] as? String else { return false }
        return state != "not-installed"
    }
}
