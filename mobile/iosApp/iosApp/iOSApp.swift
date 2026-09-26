import SwiftUI
import ComposeApp

@main
struct iOSApp: App {
    var body: some Scene {
        WindowGroup {
            ComposeView()
                .ignoresSafeArea()
        }
    }
}

struct ComposeView: UIViewControllerRepresentable {
    func makeUIViewController(context: Context) -> UIViewController {
        let baseUrl = Bundle.main.object(forInfoDictionaryKey: "ApiBaseUrl") as? String ?? "http://localhost:8080"
        return MainViewControllerKt.MainViewController(apiBaseUrl: baseUrl)
    }

    func updateUIViewController(_ uiViewController: UIViewController, context: Context) {}
}
