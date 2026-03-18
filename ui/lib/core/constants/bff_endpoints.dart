/// BFF API endpoint constants.
///
/// All BFF endpoint paths are defined here for centralized management.
/// Authentication is handled directly via OIDC (openid_client) — not
/// through the BFF. The BFF only validates JWT tokens.
library;

/// BFF API endpoints
class BffEndpoints {
  BffEndpoints._();

  /// Base path for UI endpoints
  static const String uiBase = '/ui';

  // Navigation
  static const String navigation = '$uiBase/navigation';

  // Pages
  static const String pages = '$uiBase/pages';
  static String page(String pageId) => '$pages/$pageId';
  static String pageData(String pageId) => '$pages/$pageId/data';

  // Forms
  static const String forms = '$uiBase/forms';
  static String form(String formId) => '$forms/$formId';
  static String formData(String formId) => '$forms/$formId/data';

  // Commands (single mutation endpoint)
  static const String commands = '$uiBase/commands';
  static String command(String commandId) => '$commands/$commandId';

  // Workflows
  static const String workflows = '$uiBase/workflows';
  static String workflow(String instanceId) => '$workflows/$instanceId';
  static String workflowStart(String workflowId) =>
      '$workflows/$workflowId/start';
  static String workflowAdvance(String instanceId) =>
      '$workflows/$instanceId/advance';
  static String workflowCancel(String instanceId) =>
      '$workflows/$instanceId/cancel';

  // Search and lookup
  static const String search = '$uiBase/search';
  static const String lookups = '$uiBase/lookups';
  static String lookup(String lookupId) => '$lookups/$lookupId';

  // File operations
  static const String upload = '$uiBase/upload';
  static const String download = '$uiBase/download';
  static String downloadFile(String fileId) => '$download/$fileId';

  // Health (public, no auth)
  static const String health = '$uiBase/health';
  static const String ready = '$uiBase/ready';
}
