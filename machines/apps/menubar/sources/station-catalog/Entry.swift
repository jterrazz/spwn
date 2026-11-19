import Foundation

@main
enum Entry {
    @MainActor
    static func main() {
        let arguments = Array(CommandLine.arguments.dropFirst())
        if arguments.contains("--export") { exit(Export.run(arguments)) }
        CatalogApp.main()
    }
}
