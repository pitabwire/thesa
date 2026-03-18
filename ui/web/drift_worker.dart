/// Drift web worker that runs SQLite in a shared web worker.
///
/// This file needs to be compiled to JS and placed in the web/ directory.
/// Run: dart compile js -O2 -o web/drift_worker.js web/drift_worker.dart
import 'package:drift/wasm.dart';

void main() => WasmDatabase.workerMainForOpen();
