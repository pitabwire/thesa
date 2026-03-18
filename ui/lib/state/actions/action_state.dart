/// Action execution state model.
library;

import 'package:freezed_annotation/freezed_annotation.dart';

part 'action_state.freezed.dart';
part 'action_state.g.dart';

/// Action execution state
@freezed
class ActionState with _$ActionState {
  const factory ActionState.idle() = _Idle;
  const factory ActionState.confirming({
    required String actionId,
    required String message,
    required bool isDestructive,
    required Map<String, dynamic> payload,
  }) = _Confirming;
  const factory ActionState.submitting() = _Submitting;
  const factory ActionState.success({
    required Map<String, dynamic> response,
    String? message,
    String? navigationUrl,
  }) = _Success;
  const factory ActionState.error({
    required String message,
    Map<String, String>? fieldErrors,
  }) = _Error;
}
