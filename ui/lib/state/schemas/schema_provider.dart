/// Schema provider (family) for data structure definitions.
///
/// Cache-first with 30-minute TTL.
/// Keep alive - schemas are shared across pages.
library;

import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../core/core.dart';
import '../core/dependencies_provider.dart';

part 'schema_provider.g.dart';

final _logger = Logger('SchemaProvider');

/// Schema provider (family - one instance per schema ID)
@Riverpod(keepAlive: true)
class SchemaData extends _$SchemaData {
  @override
  Future<Schema> build(String schemaId) async {
    _logger.info('Loading schema: $schemaId');

    final cacheCoordinator = await ref.read(cacheCoordinatorProvider.future);
    final bffClient = ref.read(bffClientProvider);

    try {
      final result = await cacheCoordinator.getSchema(
        schemaId,
        fetchFromNetwork: () => bffClient.getSchema(schemaId),
      );

      _logger.info(
        'Schema loaded: ${result.state.name} '
        '(${result.data.fields.length} fields)',
      );

      return result.data;
    } catch (e, stack) {
      _logger.severe('Failed to load schema: $schemaId', e, stack);
      rethrow;
    }
  }

  /// Refresh schema from server
  Future<void> refresh() async {
    _logger.info('Refreshing schema: $schemaId');
    ref.invalidateSelf();
  }

  @override
  void dispose() {
    _logger.fine('Disposing schema provider: $schemaId');
    super.dispose();
  }
}
