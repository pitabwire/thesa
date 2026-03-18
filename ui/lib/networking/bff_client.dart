/// BFF client interface for all API endpoints.
///
/// This is a manual implementation until retrofit_generator is compatible.
/// Once compatible, this will be replaced with @RestApi annotated interface.
library;

import 'package:dio/dio.dart';

import '../core/core.dart';

/// BFF API client
class BffClient {
  BffClient(this._dio);

  final Dio _dio;

  // ========== Capabilities ==========

  /// Get application capabilities
  Future<Capabilities> getCapabilities() async {
    final response = await _dio.get<Map<String, dynamic>>('/ui/capabilities');
    return Capabilities.fromJson(response.data!);
  }

  // ========== Navigation ==========

  /// Get navigation tree
  Future<NavigationTree> getNavigation() async {
    final response = await _dio.get<Map<String, dynamic>>('/ui/navigation');
    return NavigationTree.fromJson(response.data!);
  }

  // ========== Pages ==========

  /// Get page descriptor by ID
  Future<PageDescriptor> getPage(String pageId) async {
    final response = await _dio.get<Map<String, dynamic>>('/ui/pages/$pageId');
    return PageDescriptor.fromJson(response.data!);
  }

  // ========== Schemas ==========

  /// Get schema by ID
  Future<Schema> getSchema(String schemaId) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/ui/schemas/$schemaId',
    );
    return Schema.fromJson(response.data!);
  }

  // ========== Resources ==========

  /// Get resource data (paginated)
  Future<Map<String, dynamic>> getResource(
    String resourceType, {
    int? page,
    int? pageSize,
    String? sortField,
    String? sortDirection,
    Map<String, dynamic>? filters,
  }) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/ui/resources/$resourceType',
      queryParameters: {
        if (page != null) 'page': page,
        if (pageSize != null) 'page_size': pageSize,
        if (sortField != null) 'sort_field': sortField,
        if (sortDirection != null) 'sort_direction': sortDirection,
        if (filters != null) ...filters,
      },
    );
    return response.data!;
  }

  /// Get single resource item by ID
  Future<Map<String, dynamic>> getResourceItem(
    String resourceType,
    String id,
  ) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/ui/resources/$resourceType/$id',
    );
    return response.data!;
  }

  // ========== Actions ==========

  /// Execute an action
  Future<Map<String, dynamic>> executeAction(
    String actionId, {
    Map<String, dynamic>? data,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/ui/actions/$actionId',
      data: data,
    );
    return response.data!;
  }

  // ========== Workflows ==========

  /// Get workflow descriptor
  Future<WorkflowDescriptor> getWorkflow(String workflowId) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/ui/workflows/$workflowId',
    );
    return WorkflowDescriptor.fromJson(response.data!);
  }

  /// Submit workflow step
  Future<Map<String, dynamic>> submitWorkflowStep(
    String workflowId,
    String stepId, {
    required Map<String, dynamic> data,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/ui/workflows/$workflowId/steps/$stepId',
      data: data,
    );
    return response.data!;
  }

  // ========== Commands ==========

  /// Execute a BFF command (single mutation endpoint)
  Future<Map<String, dynamic>> executeCommand(
    String commandId, {
    Map<String, dynamic>? input,
    Map<String, dynamic>? routeParams,
    String? idempotencyKey,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/ui/commands/$commandId',
      data: {
        if (input != null) 'input': input,
        if (routeParams != null) 'route_params': routeParams,
        if (idempotencyKey != null) 'idempotency_key': idempotencyKey,
      },
    );
    return response.data!;
  }

  // ========== Forms ==========

  /// Get form descriptor
  Future<Map<String, dynamic>> getForm(String formId) async {
    final response = await _dio.get<Map<String, dynamic>>('/ui/forms/$formId');
    return response.data!;
  }

  /// Get form pre-fill data
  Future<Map<String, dynamic>> getFormData(String formId) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/ui/forms/$formId/data',
    );
    return response.data!;
  }

  // ========== Page Data ==========

  /// Get page data (table rows, etc.)
  Future<Map<String, dynamic>> getPageData(
    String pageId, {
    int? page,
    int? pageSize,
    String? sort,
    String? sortDir,
    Map<String, dynamic>? filters,
  }) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/ui/pages/$pageId/data',
      queryParameters: {
        if (page != null) 'page': page,
        if (pageSize != null) 'page_size': pageSize,
        if (sort != null) 'sort': sort,
        if (sortDir != null) 'sort_dir': sortDir,
        if (filters != null) ...filters,
      },
    );
    return response.data!;
  }

  // ========== Search ==========

  /// Search resources
  Future<Map<String, dynamic>> search(
    String resourceType,
    String query, {
    int? limit,
  }) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/ui/resources/$resourceType/search',
      queryParameters: {'q': query, if (limit != null) 'limit': limit},
    );
    return response.data!;
  }

  // ========== File Operations ==========

  /// Upload file
  Future<Map<String, dynamic>> uploadFile(
    String filePath, {
    void Function(int, int)? onProgress,
  }) async {
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(filePath),
    });

    final response = await _dio.post<Map<String, dynamic>>(
      '/ui/upload',
      data: formData,
      onSendProgress: onProgress,
    );
    return response.data!;
  }

  /// Download file
  Future<void> downloadFile(
    String fileId,
    String savePath, {
    void Function(int, int)? onProgress,
  }) async {
    await _dio.download(
      '/ui/download/$fileId',
      savePath,
      onReceiveProgress: onProgress,
    );
  }
}
